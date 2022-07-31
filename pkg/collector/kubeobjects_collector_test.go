package collector

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestKubeObjectsCollectorGetName(t *testing.T) {
	const expectedName = "kubeobjects"

	c := NewKubeObjectsCollector(nil, nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestKubeObjectsCollectorCheckSupported(t *testing.T) {
	c := NewKubeObjectsCollector(nil, nil)
	err := c.CheckSupported()
	if err != nil {
		t.Errorf("error checking supported: %v", err)
	}
}

var defaultKubeObjects = []string{"kube-system/pod", "kube-system/service", "kube-system/deployment"}

func getDefaultKubeObjectResults(fixture *test.ClusterFixture) (map[string]*regexp.Regexp, error) {
	results := map[string]*regexp.Regexp{}

	podList, err := fixture.AdminAccess.Clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing pods: %w", err)
	}

	for _, pod := range podList.Items {
		key := fmt.Sprintf("kube-system_pod_%s", pod.Name)
		results[key] = regexp.MustCompile(fmt.Sprintf(`^Name:\s+%s\n(.*\n)*Containers:\n(.*\n)*Conditions:\n(.*\n)*Events:`, pod.Name))
	}

	svcList, err := fixture.AdminAccess.Clientset.CoreV1().Services("kube-system").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing services: %w", err)
	}

	for _, svc := range svcList.Items {
		key := fmt.Sprintf("kube-system_service_%s", svc.Name)
		results[key] = regexp.MustCompile(fmt.Sprintf(`^Name:\s+%s\n(.*\n)*Type:(.*\n)*IP:(.*\n)*Endpoints:(.*\n)*Events:`, svc.Name))
	}

	deployList, err := fixture.AdminAccess.Clientset.AppsV1().Deployments("kube-system").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing deployments: %w", err)
	}

	for _, deploy := range deployList.Items {
		key := fmt.Sprintf("kube-system_deployment_%s", deploy.Name)
		results[key] = regexp.MustCompile(fmt.Sprintf(`^Name:\s+%s\n(.*\n)*Pod Template:\n(.*\n)*Conditions:\n(.*\n)*Events:`, deploy.Name))
	}

	return results, nil
}

func getNodeNames(fixture *test.ClusterFixture) ([]string, error) {
	nodeList, err := fixture.AdminAccess.Clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing nodes: %w", err)
	}

	nodeNames := make([]string, len(nodeList.Items))
	for i, node := range nodeList.Items {
		nodeNames[i] = node.Name
	}

	return nodeNames, nil
}

func getNodeResults(nodeNames []string) map[string]*regexp.Regexp {
	results := map[string]*regexp.Regexp{}
	for _, nodeName := range nodeNames {
		key := fmt.Sprintf("_nodes_%s", nodeName)
		results[key] = regexp.MustCompile(fmt.Sprintf(`^Name:\s+%s\n(.*\n)*Conditions:\n(.*\n)*System Info:\n(.*\n)*Events:`, nodeName))
	}

	return results
}

func TestKubeObjectsCollectorCollect(t *testing.T) {
	fixture, _ := test.GetClusterFixture()

	testNamespace, err := fixture.CreateTestNamespace("kubeobjectstest")
	if err != nil {
		t.Fatalf("Error creating test namespace %s: %v", testNamespace, err)
	}

	deployResourcesCommand := fmt.Sprintf("kubectl apply -n %s -f /resources/kube-objects/test-resources.yaml", testNamespace)
	_, err = fixture.CommandRunner.Run(deployResourcesCommand, fixture.AdminAccess.GetKubeConfigBinding())
	if err != nil {
		t.Fatalf("Error deploying test resources into %s namespace: %v", testNamespace, err)
	}

	defaultResults, err := getDefaultKubeObjectResults(fixture)
	if err != nil {
		t.Fatalf("Error determining expected results for default configuration: %v", err)
	}

	nodeNames, err := getNodeNames(fixture)
	if err != nil {
		t.Fatalf("Error getting node names: %v", err)
	}

	tests := []struct {
		name             string
		requestedObjects []string
		config           *rest.Config
		wantErr          bool
		want             map[string]*regexp.Regexp
	}{
		{
			name:             "bad kubeconfig",
			requestedObjects: defaultKubeObjects,
			config:           &rest.Config{Host: string([]byte{0})},
			wantErr:          true,
			want:             nil,
		},
		{
			name:             "too few kubeobject parts should be skipped",
			requestedObjects: []string{"kube-system"},
			config:           fixture.PeriscopeAccess.ClientConfig,
			wantErr:          false,
			want:             map[string]*regexp.Regexp{},
		},
		{
			name:             "unknown resource type should be skipped",
			requestedObjects: []string{"kube-system/notaresource"},
			config:           fixture.PeriscopeAccess.ClientConfig,
			wantErr:          false,
			want:             map[string]*regexp.Regexp{},
		},
		{
			name:             "undescribable resource type should be skipped",
			requestedObjects: []string{"kube-system/events"},
			config:           fixture.PeriscopeAccess.ClientConfig,
			wantErr:          false,
			want:             map[string]*regexp.Regexp{},
		},
		{
			name:             "missing resource should be skipped",
			requestedObjects: []string{"kube-system/pod/notexisting"},
			config:           fixture.PeriscopeAccess.ClientConfig,
			wantErr:          false,
			want:             map[string]*regexp.Regexp{},
		},
		{
			name:             "unknown namespace should be skipped",
			requestedObjects: []string{"notanamespace/pod"},
			config:           fixture.PeriscopeAccess.ClientConfig,
			wantErr:          false,
			want:             map[string]*regexp.Regexp{},
		},
		{
			name:             "non-namespaced resource type can be described",
			requestedObjects: []string{"/nodes"},
			config:           fixture.PeriscopeAccess.ClientConfig,
			wantErr:          false,
			want:             getNodeResults(nodeNames),
		},
		{
			name:             "single non-namespaced resource can be described",
			requestedObjects: []string{fmt.Sprintf("/nodes/%s", nodeNames[0])},
			config:           fixture.PeriscopeAccess.ClientConfig,
			wantErr:          false,
			want:             getNodeResults([]string{nodeNames[0]}),
		},
		{
			name:             "specified resources",
			requestedObjects: []string{fmt.Sprintf("%s/configmaps", testNamespace)},
			config:           fixture.PeriscopeAccess.ClientConfig,
			wantErr:          false,
			want: map[string]*regexp.Regexp{
				fmt.Sprintf("%s_configmaps_kube-root-ca.crt", testNamespace): regexp.MustCompile(`^Name:\s+kube-root-ca.crt\n(.*\n)*Data`),
				fmt.Sprintf("%s_configmaps_test-configmap-1", testNamespace): regexp.MustCompile(`^Name:\s+test-configmap-1\n(.*\n)*Data`),
				fmt.Sprintf("%s_configmaps_test-configmap-2", testNamespace): regexp.MustCompile(`^Name:\s+test-configmap-2\n(.*\n)*Data`),
				fmt.Sprintf("%s_configmaps_test-configmap-3", testNamespace): regexp.MustCompile(`^Name:\s+test-configmap-3\n(.*\n)*Data`),
			},
		},
		{
			name:             "single resource",
			requestedObjects: []string{fmt.Sprintf("%s/configmaps/test-configmap-2", testNamespace)},
			config:           fixture.PeriscopeAccess.ClientConfig,
			wantErr:          false,
			want: map[string]*regexp.Regexp{
				fmt.Sprintf("%s_configmaps_test-configmap-2", testNamespace): regexp.MustCompile(`^Name:\s+test-configmap-2\n(.*\n)*Data`),
			},
		},
		{
			name:             "default kubeobjects",
			requestedObjects: defaultKubeObjects,
			config:           fixture.PeriscopeAccess.ClientConfig,
			wantErr:          false,
			want:             defaultResults,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeInfo := &utils.RuntimeInfo{
				KubernetesObjects: tt.requestedObjects,
			}

			c := NewKubeObjectsCollector(tt.config, runtimeInfo)

			err := c.Collect()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but none found")
				}
				return
			}

			data := c.GetData()

			compareCollectorData(t, tt.want, data)
		})
	}
}
