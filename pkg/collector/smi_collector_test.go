package collector

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestSmiCollectorGetName(t *testing.T) {
	const expectedName = "smi"

	c := NewSmiCollector(nil, nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestSmiCollectorCheckSupported(t *testing.T) {
	tests := []struct {
		name       string
		collectors []string
		wantErr    bool
	}{
		{
			name:       "no OSM or SMI included",
			collectors: []string{"NOT_OSM", "NOT_SMI"},
			wantErr:    true,
		},
		{
			name:       "only OSM included",
			collectors: []string{"OSM", "NOT_SMI"},
			wantErr:    false,
		},
		{
			name:       "only SMI included",
			collectors: []string{"NOT_OSM", "SMI"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		runtimeInfo := &utils.RuntimeInfo{
			CollectorList: tt.collectors,
		}
		c := NewSmiCollector(nil, runtimeInfo)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckSupported() for %s error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestSmiCollectorCollect(t *testing.T) {
	fixture, _ := test.GetClusterFixture()

	tests := []struct {
		name    string
		want    map[string]*regexp.Regexp
		wantErr bool
	}{
		{
			name:    "SMI deployments found",
			want:    getExpectedSmiData(fixture.KnownNamespaces),
			wantErr: false,
		},
	}

	runtimeInfo := &utils.RuntimeInfo{
		CollectorList: []string{"SMI"},
	}

	c := NewSmiCollector(fixture.PeriscopeAccess.ClientConfig, runtimeInfo)

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

type crdResource struct {
	namespace  string
	name       string
	apiVersion string
}

func getExpectedSmiData(knownNamespaces *test.KnownNamespaces) map[string]*regexp.Regexp {
	crdResourceGroups := []struct {
		crdName   string
		kind      string
		resources []crdResource
	}{
		{
			crdName: "traffictargets.access.smi-spec.io",
			kind:    "TrafficTarget",
			resources: []crdResource{
				{
					namespace:  knownNamespaces.OsmBookStore,
					name:       "bookstore",
					apiVersion: "access.smi-spec.io/v1alpha3",
				},
				{
					namespace:  knownNamespaces.OsmBookStore,
					name:       "bookstore-v2",
					apiVersion: "access.smi-spec.io/v1alpha3",
				},
				{
					namespace:  knownNamespaces.OsmBookWarehouse,
					name:       "mysql",
					apiVersion: "access.smi-spec.io/v1alpha3",
				},
				{
					namespace:  knownNamespaces.OsmBookWarehouse,
					name:       "bookstore-access-bookwarehouse",
					apiVersion: "access.smi-spec.io/v1alpha3",
				},
			},
		},
		{
			crdName: "httproutegroups.specs.smi-spec.io",
			kind:    "HTTPRouteGroup",
			resources: []crdResource{
				{
					namespace:  knownNamespaces.OsmBookStore,
					name:       "bookstore-service-routes",
					apiVersion: "specs.smi-spec.io/v1alpha4",
				},
				{
					namespace:  knownNamespaces.OsmBookWarehouse,
					name:       "bookwarehouse-service-routes",
					apiVersion: "specs.smi-spec.io/v1alpha4",
				},
			},
		},
		{
			crdName: "tcproutes.specs.smi-spec.io",
			kind:    "TCPRoute",
			resources: []crdResource{
				crdResource{
					namespace:  knownNamespaces.OsmBookWarehouse,
					name:       "mysql",
					apiVersion: "specs.smi-spec.io/v1alpha4",
				},
			},
		},
		{
			crdName: "trafficsplits.split.smi-spec.io",
			kind:    "TrafficSplit",
			resources: []crdResource{
				crdResource{
					namespace:  knownNamespaces.OsmBookStore,
					name:       "bookstore-split",
					apiVersion: "split.smi-spec.io/v1alpha2",
				},
			},
		},
	}

	result := map[string]*regexp.Regexp{}
	for _, crdResourceGroup := range crdResourceGroups {
		// Expect the key to be based on CRD with .io suffix removed.
		key := fmt.Sprintf("smi/crd_%s", strings.TrimSuffix(crdResourceGroup.crdName, ".io"))

		// Expect the value to be a yaml CRD with the right `name` and `kind` values.
		result[key] = regexp.MustCompile(fmt.Sprintf(
			`^apiVersion: apiextensions\.k8s\.io/v1\nkind: CustomResourceDefinition\n(.*\n)*  name: %s\n(.*\n)*    kind: %s`,
			crdResourceGroup.crdName, crdResourceGroup.kind))

		for _, resource := range crdResourceGroup.resources {
			// Expect the key to be composed of namespace, CRD and resource name.
			key := fmt.Sprintf("smi/namespace_%s/%s_%s_custom_resource", resource.namespace, crdResourceGroup.crdName, resource.name)

			// Expect the value to be a yaml custom resource with the right `apiVersion`, `kind`, `name` and `namespace` values.
			result[key] = regexp.MustCompile(fmt.Sprintf(`^apiVersion: %s\nkind: %s\n(.*\n)*  name: %s\n  namespace: %s`,
				resource.apiVersion, crdResourceGroup.kind, resource.name, resource.namespace))
		}
	}

	return result
}
