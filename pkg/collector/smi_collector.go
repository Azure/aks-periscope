package collector

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/Azure/aks-periscope/pkg/utils"
)

// SmiCollector defines an Smi Collector struct
type SmiCollector struct {
	data map[string]string
}

// NewSmiCollector is a constructor
func NewSmiCollector() *SmiCollector {
	return &SmiCollector{
		data: make(map[string]string),
	}
}

func (collector *SmiCollector) GetName() string {
	return "smi"
}

func (collector *SmiCollector) CheckSupported() error {
	// This is not currently supported on Windows because it launches `kubectl` as a separate process (within GetResourceList).
	// If/when it is reimplemented using the go client API for k8s, we can re-enable this.
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
	}

	return nil
}

func (collector *SmiCollector) GetData() map[string]string {
	return collector.data
}

// Collect implements the interface method
func (collector *SmiCollector) Collect() error {
	// Get all CustomResourceDefinitions in the cluster
	allCrdsList, err := utils.GetResourceList([]string{"get", "crds", "-o", "jsonpath={..metadata.name}"}, " ")
	if err != nil {
		return err
	}

	// Filter to obtain a list of Smi CustomResourceDefinitions in the cluster
	var smiCrdsList []string
	for _, s := range allCrdsList {
		if strings.Contains(s, "smi-spec.io") {
			smiCrdsList = append(smiCrdsList, s)
		}
	}
	if len(smiCrdsList) == 0 {
		return errors.New("cluster does not contain any SMI CRDs")
	}

	collectSmiCrds(collector, smiCrdsList)
	return collectSmiCustomResourcesFromAllNamespaces(collector, smiCrdsList)
}

func collectSmiCrds(collector *SmiCollector, smiCrdsList []string) {
	for _, smiCrd := range smiCrdsList {
		yamlDefinition, err := utils.RunCommandOnContainer("kubectl", "get", "crd", smiCrd, "-o", "yaml")
		if err != nil {
			log.Printf("Skipping: unable to collect yaml definition of %s: %+v", smiCrd, err)
		}
		collector.data["smi/crd_"+strings.TrimSuffix(smiCrd, ".io")] = yamlDefinition
	}
}

func collectSmiCustomResourcesFromAllNamespaces(collector *SmiCollector, smiCrdsList []string) error {
	// Get all namespaces in the cluster
	namespacesList, err := utils.GetResourceList([]string{"get", "namespaces", "-o", "jsonpath={..metadata.name}"}, " ")
	if err != nil {
		return fmt.Errorf("collect SMI custom resources: unable to list namespaces in the cluster: %w", err)
	}

	for _, namespace := range namespacesList {
		collectSmiCustomResourcesFromNamespace(collector, smiCrdsList, namespace)
	}

	return nil
}

func collectSmiCustomResourcesFromNamespace(collector *SmiCollector, smiCrdsList []string, namespace string) {
	for _, smiCrdType := range smiCrdsList {
		// get all custom resources of this smi crd type
		customResourcesList, err := utils.GetResourceList([]string{"get", smiCrdType, "-n", namespace, "-o", "jsonpath={..metadata.name}"}, " ")
		if err != nil {
			log.Printf("Skipping: unable to list custom resources of type %s in namespace %s: %+v", smiCrdType, namespace, err)
			continue
		}

		for _, customResourceName := range customResourcesList {
			yamlDefinition, err := utils.RunCommandOnContainer("kubectl", "get", smiCrdType, customResourceName, "-n", namespace, "-o", "yaml")
			if err != nil {
				log.Printf("Skipping: unable to collect yaml definition of %s (custom resource type: %s): %+v", customResourceName, smiCrdType, err)
			}
			collector.data["smi/namespace_"+namespace+"/"+smiCrdType+"_"+customResourceName+"_custom_resource"] = yamlDefinition
		}
	}
}
