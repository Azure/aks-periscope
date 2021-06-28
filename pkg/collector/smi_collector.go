package collector

import (
	"fmt"
	"log"
	"path/filepath"
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

// Collect implements the interface method
func (collector *SmiCollector) Collect() error {
	// Get all CustomResourceDefinitions in the cluster
	allCrdsList, err := utils.GetResourceList([]string{"get", "crds", "-o", "jsonpath={..metadata.name}"}, " ")
	if err != nil {
		return err
	}

	// Directory where logs will be written to
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	// Filter to obtain a list of Smi CustomResourceDefinitions in the cluster
	crdNameContainsSmiPredicate := func(s string) bool { return strings.Contains(s, "smi-spec.io") }
	smiCrdsList := filter(allCrdsList, crdNameContainsSmiPredicate)
	if len(smiCrdsList) == 0 {
		return fmt.Errorf("Cluster does not contain any SMI CRDs")
	}

	collectSmiCrds(collector, filepath.Join(rootPath, "smi_crd_definitions"), smiCrdsList)
	return collectSmiCustomResourcesFromAllNamespaces(collector, filepath.Join(rootPath, "smi_custom_resources"), smiCrdsList)
}

func collectSmiCrds(collector *SmiCollector, rootPath string, smiCrdsList []string) {
	for _, smiCrd := range smiCrdsList {
		fileName := smiCrd + "_definition.yaml"
		kubeCmd := []string{"get", "crd", smiCrd, "-o", "yaml"}
		if err := collector.CollectKubectlOutputToCollectorFiles(rootPath, fileName, kubeCmd); err != nil {
			log.Printf("Skipping: unable to collect yaml definition of %s to %s: %+v", smiCrd, fileName, err)
		}
	}
}

func collectSmiCustomResourcesFromAllNamespaces(collector *SmiCollector, rootPath string, smiCrdsList []string) error {
	// Get all namespaces in the cluster
	namespacesList, err := utils.GetResourceList([]string{"get", "namespaces", "-o", "jsonpath={..metadata.name}"}, " ")
	if err != nil {
		return fmt.Errorf("Failed to collect SMI custom resources: unable to list namespaces in the cluster: %+v", err)
	}

	for _, namespace := range namespacesList {
		// all SMI custom resources in the namespace will be collected to the directory "namespace_X" where X is the namespace name
		namespaceRootPath := filepath.Join(rootPath, "namespace_"+namespace)
		collectSmiCustomResourcesFromNamespace(collector, namespaceRootPath, smiCrdsList, namespace)
	}

	return nil
}

func collectSmiCustomResourcesFromNamespace(collector *SmiCollector, rootPath string, smiCrdsList []string, namespace string) {
	for _, smiCrdType := range smiCrdsList {
		// get all custom resources of this smi crd type
		customResourcesList, err := utils.GetResourceList([]string{"get", smiCrdType, "-n", namespace, "-o", "jsonpath={..metadata.name}"}, " ")
		if err != nil {
			log.Printf("Skipping: unable to list custom resources of type %s in namespace %s: %+v", smiCrdType, namespace, err)
			continue
		}

		customResourcesRootPath := filepath.Join(rootPath, smiCrdType+"_custom_resources")
		for _, customResourceName := range customResourcesList {
			fileName := smiCrdType + "_" + customResourceName + ".yaml"
			kubeCmd := []string{"get", smiCrdType, customResourceName, "-n", namespace, "-o", "yaml"}
			if err := collector.CollectKubectlOutputToCollectorFiles(customResourcesRootPath, fileName, kubeCmd); err != nil {
				log.Printf("Skipping: unable to collect yaml definition of %s (custom resource type: %s) to %s: %+v", customResourceName, smiCrdType, fileName, err)
			}
		}
	}
}

func filter(ss []string, test func(string) bool) (ret []string) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}
