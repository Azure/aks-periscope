package collector

import (
	"fmt"
	"strings"

	"github.com/Azure/aks-periscope/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

// SmiCollector defines an Smi Collector struct
type SmiCollector struct {
	data          map[string]string
	kubeconfig    *rest.Config
	commandRunner *utils.KubeCommandRunner
	runtimeInfo   *utils.RuntimeInfo
}

// NewSmiCollector is a constructor
func NewSmiCollector(config *rest.Config, runtimeInfo *utils.RuntimeInfo) *SmiCollector {
	return &SmiCollector{
		data:          make(map[string]string),
		kubeconfig:    config,
		commandRunner: utils.NewKubeCommandRunner(config),
		runtimeInfo:   runtimeInfo,
	}
}

func (collector *SmiCollector) GetName() string {
	return "smi"
}

func (collector *SmiCollector) CheckSupported() error {
	if !utils.Contains(collector.runtimeInfo.CollectorList, "OSM") && !utils.Contains(collector.runtimeInfo.CollectorList, "SMI") {
		return fmt.Errorf("not included because neither 'OSM' or 'SMI' are in COLLECTOR_LIST variable. Included values: %s", strings.Join(collector.runtimeInfo.CollectorList, " "))
	}

	return nil
}

func (collector *SmiCollector) GetData() map[string]string {
	return collector.data
}

// Collect implements the interface method
func (collector *SmiCollector) Collect() error {
	smiCrds, err := collector.getAllSmiCrds()
	if err != nil {
		return fmt.Errorf("error getting SMI CRDs: %w", err)
	}

	// Store the CRD definitions as collector data
	for _, smiCrd := range smiCrds {
		trimmedName := strings.TrimSuffix(smiCrd.GetName(), ".io")
		yaml, err := collector.commandRunner.PrintAsYaml(&smiCrd)
		if err != nil {
			return fmt.Errorf("error printing CRD %s as YAML: %w", trimmedName, err)
		}
		key := fmt.Sprintf("smi/crd_%s", trimmedName)
		collector.data[key] = yaml
	}

	// Get the GroupVersionResource identifiers for all the resources for these CRDs
	gvrs := []schema.GroupVersionResource{}
	for _, smiCrd := range smiCrds {
		gvr, err := collector.commandRunner.GetGVRFromCRD(&smiCrd)
		if err != nil {
			return fmt.Errorf("error getting GVR from CRD %s: %+v", smiCrd.GetName(), err)
		}
		gvrs = append(gvrs, *gvr)
	}

	// Get the resources in all the namespaces for all possible versions of all the CRDs.
	smiResources, err := collector.getSmiCustomResourcesFromAllNamespaces(gvrs)
	if err != nil {
		return fmt.Errorf("error getting custom SMI resources for all namespaces: %w", err)
	}

	// Store the resource definitions as collector data
	for _, resource := range smiResources {
		crdName := resource.GroupResource().String() // e.g. "traffictargets.access.smi-spec.io"
		key := fmt.Sprintf("smi/namespace_%s/%s_%s_custom_resource", resource.namespace, crdName, resource.name)
		collector.data[key] = resource.yaml
	}

	return nil
}

type smiResource struct {
	namespace string
	schema.GroupVersionResource
	name string
	yaml string
}

func (collector *SmiCollector) getAllSmiCrds() ([]unstructured.Unstructured, error) {
	// Get all the CRDs in the cluster (we'll filter them according to a pattern, so can't retrieve them by name).
	crds, err := collector.commandRunner.GetCRDUnstructuredList()
	if err != nil {
		return nil, fmt.Errorf("error listing CRDs in cluster")
	}

	results := []unstructured.Unstructured{}
	for _, crd := range crds.Items {
		if strings.Contains(crd.GetName(), "smi-spec.io") {
			results = append(results, crd)
		}
	}

	return results, nil
}

func (collector *SmiCollector) getSmiCustomResourcesFromAllNamespaces(gvrs []schema.GroupVersionResource) ([]smiResource, error) {
	result := []smiResource{}
	for _, gvr := range gvrs {
		// Find resources in all namespaces
		resources, err := collector.commandRunner.GetUnstructuredList(&gvr, "", &metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("error listing %s resources: %w", gvr.String(), err)
		}
		for _, resource := range resources.Items {
			yaml, err := collector.commandRunner.PrintAsYaml(&resource)
			if err != nil {
				return nil, fmt.Errorf("error getting yaml for %s: %w", resource.GetName(), err)
			}
			result = append(result, smiResource{
				namespace:            resource.GetNamespace(),
				GroupVersionResource: gvr,
				name:                 resource.GetName(),
				yaml:                 yaml,
			})
		}
	}

	return result, nil
}
