package collector

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestOsmCollectorGetName(t *testing.T) {
	const expectedName = "osm"

	c := NewOsmCollector(nil, nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestOsmCollectorCheckSupported(t *testing.T) {
	tests := []struct {
		name       string
		collectors []string
		wantErr    bool
	}{
		{
			name:       "OSM not included",
			collectors: []string{"NOT_OSM"},
			wantErr:    true,
		},
		{
			name:       "OSM included",
			collectors: []string{"OSM"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		runtimeInfo := &utils.RuntimeInfo{
			CollectorList: tt.collectors,
		}
		c := NewOsmCollector(nil, runtimeInfo)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckSupported() for %s error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func setupOsmTest(t *testing.T) *test.ClusterFixture {
	fixture, _ := test.GetClusterFixture()

	// Run commands to wait until the OSM application rollout is complete
	commands := []string{
		fmt.Sprintf("kubectl rollout status -n %s deploy/bookbuyer --timeout=240s", fixture.KnownNamespaces.OsmBookBuyer),
		fmt.Sprintf("kubectl rollout status -n %s deploy/bookthief --timeout=240s", fixture.KnownNamespaces.OsmBookThief),
		fmt.Sprintf("kubectl rollout status -n %s deploy/bookstore --timeout=240s", fixture.KnownNamespaces.OsmBookStore),
		fmt.Sprintf("kubectl rollout status -n %s deploy/bookstore-v2 --timeout=240s", fixture.KnownNamespaces.OsmBookStore),
		fmt.Sprintf("kubectl rollout status -n %s deploy/bookwarehouse --timeout=240s", fixture.KnownNamespaces.OsmBookWarehouse),
		fmt.Sprintf("kubectl rollout status -n %s statefulset/mysql --timeout=240s", fixture.KnownNamespaces.OsmBookWarehouse),
	}

	_, err := fixture.CommandRunner.Run(strings.Join(commands, " && "), fixture.AdminAccess.GetKubeConfigBinding())
	if err != nil {
		t.Fatalf("error waiting for OSM application rollout to complete: %v", err)
	}

	return fixture
}

func TestOsmCollectorCollect(t *testing.T) {
	fixture := setupOsmTest(t)

	expectedData, err := getExpectedOsmData(fixture.AdminAccess.Clientset, fixture.KnownNamespaces)
	if err != nil {
		t.Fatalf("unable to get expected OSM data keys: %v", err)
	}

	tests := []struct {
		name    string
		want    map[string]*regexp.Regexp
		wantErr bool
	}{
		{
			name:    "OSM deployments found",
			want:    expectedData,
			wantErr: false,
		},
	}

	runtimeInfo := &utils.RuntimeInfo{
		CollectorList: []string{"OSM"},
	}

	c := NewOsmCollector(fixture.PeriscopeAccess.ClientConfig, runtimeInfo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()

			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}
			data := c.GetData()

			missingDataKeys := []string{}
			for key, regexp := range tt.want {
				value, ok := data[key]
				if ok {
					if !regexp.MatchString(value) {
						t.Errorf("unexpected value for %s\n\texpected: %s\n\tfound: %s", key, regexp.String(), value)
					}
				} else {
					missingDataKeys = append(missingDataKeys, key)
				}
			}
			if len(missingDataKeys) > 0 {
				t.Errorf("missing keys in Collect result:\n%s", strings.Join(missingDataKeys, "\n"))
			}

			unexpectedDataKeys := []string{}
			for key := range data {
				if _, ok := tt.want[key]; !ok {
					unexpectedDataKeys = append(unexpectedDataKeys, key)
				}
			}
			if len(unexpectedDataKeys) > 0 {
				t.Errorf("unexpected keys in Collect result:\n%s", strings.Join(unexpectedDataKeys, "\n"))
			}
		})
	}
}

func getExpectedOsmData(clientset *kubernetes.Clientset, knownNamespaces *test.KnownNamespaces) (map[string]*regexp.Regexp, error) {
	const meshName = "test-osm"

	applicationNamespaces := []string{
		knownNamespaces.OsmBookBuyer,
		knownNamespaces.OsmBookStore,
		knownNamespaces.OsmBookThief,
		knownNamespaces.OsmBookWarehouse,
	}

	allRelevantNamespaces := append(applicationNamespaces, knownNamespaces.OsmSystem)

	// Get the pod names in each namespace
	namespacePods := map[string][]string{}
	for _, namespace := range allRelevantNamespaces {
		podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("error listing pods in namespace %s: %w", namespace, err)
		}

		podNames := make([]string, 0, len(podList.Items))
		for _, pod := range podList.Items {
			podNames = append(podNames, pod.Name)
		}

		namespacePods[namespace] = podNames
	}

	result := map[string]*regexp.Regexp{
		// all_resources_list will be listed in tabular format
		fmt.Sprintf("%s/control_plane/all_resources_list", meshName): regexp.MustCompile(`^(NAMESPACE\s+NAME\s.*\n(.*\n)*){4}`),
		// all_resources_configs will be a json list of items
		fmt.Sprintf("%s/control_plane/all_resources_configs", meshName): regexp.MustCompile(`^{\n    "apiVersion": "v1",\n    "items": \[\n`),
		// mutating_webhook_configurations will be a json list of MutatingWebhookConfiguration items
		fmt.Sprintf("%s/control_plane/mutating_webhook_configurations", meshName): regexp.MustCompile(`^{\n    "apiVersion": "v1",\n\s+"items": \[\n\s+{\n\s+"apiVersion": "admissionregistration\.k8s\.io/v1",\n\s+"kind": "MutatingWebhookConfiguration",\n`),
		// validating_webhook_configurations will be a json list of ValidatingWebhookConfiguration items
		fmt.Sprintf("%s/control_plane/validating_webhook_configurations", meshName): regexp.MustCompile(`^{\n    "apiVersion": "v1",\n\s+"items": \[\n\s+{\n\s+"apiVersion": "admissionregistration\.k8s\.io/v1",\n\s+"kind": "ValidatingWebhookConfiguration",\n`),
		// mesh_configs will be a json list of MeshConfig items (of unspecified apiVersion config.openservicemesh.io/<version>)
		fmt.Sprintf("%s/control_plane/mesh_configs", meshName): regexp.MustCompile(`^{\n    "apiVersion": "v1",\n\s+"items": \[\n\s+{\n\s+"apiVersion": "config\.openservicemesh\.io/\w+",\n\s+"kind": "MeshConfig",\n`),
	}

	perNamespaceItems := map[string](func(string) *regexp.Regexp){
		// metadata will be the json namespace
		"metadata": func(namespace string) *regexp.Regexp {
			return regexp.MustCompile(fmt.Sprintf(`^{\n    "apiVersion": "v1",\n    "kind": "Namespace",\n(.*\n)*        "name": "%s"`, namespace))
		},
		// services_list will be in tabilar format
		"services_list": func(namespace string) *regexp.Regexp {
			if namespace == knownNamespaces.OsmBookBuyer || namespace == knownNamespaces.OsmBookThief {
				// No services for these namespaces
				return regexp.MustCompile(`^$`)
			}
			return regexp.MustCompile(`^NAME\s+TYPE\s+CLUSTER-IP\s+EXTERNAL-IP\s+PORT\(S\)\s+AGE\s+SELECTOR\n`)
		},
		// services will be a json list of Service items in the relevant namespace
		"services": func(namespace string) *regexp.Regexp {
			if namespace == knownNamespaces.OsmBookBuyer || namespace == knownNamespaces.OsmBookThief {
				// No services for these namespaces
				return regexp.MustCompile(`^{\n    "apiVersion": "v1",\n\s+"items": \[\],\n`)
			}
			return regexp.MustCompile(fmt.Sprintf(`^{\n    "apiVersion": "v1",\n\s+"items": \[\n\s+{\n\s+"apiVersion": "v1",\n\s+"kind": "Service",\n(.*\n)*\s*"namespace": "%s"`, namespace))
		},
		// endpoints_list will be in tabilar format
		"endpoints_list": func(namespace string) *regexp.Regexp {
			if namespace == knownNamespaces.OsmBookBuyer || namespace == knownNamespaces.OsmBookThief {
				// No endpoints for these namespaces
				return regexp.MustCompile(`^$`)
			}
			return regexp.MustCompile(`^NAME\s+ENDPOINTS\s+AGE\n`)
		},
		// endpoints will be a json list of Endpoints items in the relevant namespace
		"endpoints": func(namespace string) *regexp.Regexp {
			if namespace == knownNamespaces.OsmBookBuyer || namespace == knownNamespaces.OsmBookThief {
				// No endpoints for these namespaces
				return regexp.MustCompile(`^{\n    "apiVersion": "v1",\n\s+"items": \[\],\n`)
			}
			return regexp.MustCompile(fmt.Sprintf(`^{\n    "apiVersion": "v1",\n\s+"items": \[\n\s+{\n\s+"apiVersion": "v1",\n\s+"kind": "Endpoints",\n(.*\n)*\s*"namespace": "%s"`, namespace))
		},
		// configmaps_list will be in tabilar format
		"configmaps_list": func(_ string) *regexp.Regexp {
			return regexp.MustCompile(`^NAME\s+DATA\s+AGE\n`)
		},
		// configmaps will be a json list of ConfigMap items in the relevant namespace
		"configmaps": func(namespace string) *regexp.Regexp {
			return regexp.MustCompile(fmt.Sprintf(`^{\n    "apiVersion": "v1",\n\s+"items": \[\n\s+{\n\s+"apiVersion": "v1",\n(.*\n)*\s+"kind": "ConfigMap",\n(.*\n)*\s*"namespace": "%s"`, namespace))
		},
		// ingresses_list would be in tabilar format, but is empty for our test cluster
		"ingresses_list": func(_ string) *regexp.Regexp {
			return regexp.MustCompile(`^$`)
		},
		// ingresses will be a json list of Ingress items in the relevant namespace, but will contain no items for our test cluster
		"ingresses": func(_ string) *regexp.Regexp {
			return regexp.MustCompile(`^{\n    "apiVersion": "v1",\n\s+"items": \[\],\n`)
		},
		// service_accounts_list will be in tabilar format
		"service_accounts_list": func(_ string) *regexp.Regexp {
			return regexp.MustCompile(`^NAME\s+SECRETS\s+AGE\n`)
		},
		// service_accounts will be a json list of ServiceAccount items in the relevant namespace
		"service_accounts": func(namespace string) *regexp.Regexp {
			return regexp.MustCompile(fmt.Sprintf(`^{\n    "apiVersion": "v1",\n\s+"items": \[\n\s+{\n\s+"apiVersion": "v1",\n\s+"kind": "ServiceAccount",\n(.*\n)*\s*"namespace": "%s"`, namespace))
		},
		// pods_list will be in tabilar format
		"pods_list": func(_ string) *regexp.Regexp {
			return regexp.MustCompile(`^NAME\s+READY\s+STATUS\s+RESTARTS\s+AGE\s+IP\s+NODE\s+NOMINATED NODE\s+READINESS GATES\n`)
		},
	}

	// Expect a key for each of the above items, for all the OSM application namespaces *and* the OSM system namespace.
	for _, namespace := range allRelevantNamespaces {
		for itemType, regexpGetter := range perNamespaceItems {
			key := fmt.Sprintf("%s/%s_%s", meshName, namespace, itemType)
			result[key] = regexpGetter(namespace)
		}
	}

	envoyQueryValues := map[string]*regexp.Regexp{
		"config_dump": regexp.MustCompile(`^{\n "configs": \[\n  {\n   "@type": "type\.googleapis\.com/envoy\.admin\.v3\.BootstrapConfigDump",\n`),
		"clusters":    regexp.MustCompile(`.+::.+::.+\n`), // double-colon-separated triplets
		"listeners":   regexp.MustCompile(``),             // not always populated
		"ready":       regexp.MustCompile(`^LIVE\n$`),
		"stats":       regexp.MustCompile(`.+: .+\n`), // colon-separated pairs
	}

	// For the application namespaces, we expect a value for each of the five envoy queries, for each pod.
	for _, namespace := range applicationNamespaces {
		for _, podName := range namespacePods[namespace] {
			for query, regexp := range envoyQueryValues {
				// TODO: Preserving existing behaviour for now, but there should probably be a separator between
				// the pod name and the query value in the key.
				key := fmt.Sprintf("%s/envoy/%s%s", meshName, podName, query)
				result[key] = regexp
			}
		}
	}

	// Expect a podConfig value for all pods in all relevant namespaces.
	for _, namespace := range allRelevantNamespaces {
		for _, podName := range namespacePods[namespace] {
			key := fmt.Sprintf("%s/%s_podConfig", meshName, podName)
			// podConfig will be the json pod spec.
			result[key] = regexp.MustCompile(fmt.Sprintf(`^{\n    "apiVersion": "v1",\n    "kind": "Pod",\n(.*\n)*\s+"name": "%s",\n\s+"namespace": "%s",\n`, podName, namespace))
		}
	}

	// Expect podLogs for all pods in the OSM system namespace (only).
	for _, podName := range namespacePods[knownNamespaces.OsmSystem] {
		key := fmt.Sprintf("%s/%s_podLogs", meshName, podName)
		// podConfig will be the json pod spec.
		result[key] = regexp.MustCompile(`^{"level":"info","component":"osm-(bootstrap|controller|injector)`)
	}

	return result, nil
}
