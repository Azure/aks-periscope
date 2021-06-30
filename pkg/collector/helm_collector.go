package collector

import (
	"github.com/Azure/aks-periscope/pkg/utils"
)

// HelmCollector defines a Helm Collector struct
type HelmCollector struct {
	data map[string]string
}

// NewHelmCollector is a constructor
func NewHelmCollector() *HelmCollector {
	return &HelmCollector{
		data: make(map[string]string),
	}
}

func (collector *HelmCollector) GetName() string {
	return "helm"
}

// Collect implements the interface method
func (collector *HelmCollector) Collect() error {
	helmList, err := utils.RunCommandOnContainer("helm", "list", "--all-namespaces")
	if err != nil {
		return err
	}

	collector.data["helm_list"] = helmList

	helmHistory, err := utils.RunCommandOnContainer("helm", "history", "-n", "default", "azure-arc")
	if err != nil {
		return err
	}

	collector.data["helm_history"] = helmHistory

	return nil
}

func (collector *HelmCollector) GetData() map[string]string {
	return collector.data
}
