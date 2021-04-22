package collector

import (
	"path/filepath"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// HelmCollector defines a Helm Collector struct
type HelmCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &HelmCollector{}

// NewHelmCollector is a constructor
func NewHelmCollector(exporter interfaces.Exporter) *HelmCollector {
	return &HelmCollector{
		BaseCollector: BaseCollector{
			collectorType: Helm,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *HelmCollector) Collect() error {
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	helmListFile := filepath.Join(rootPath, "helm_list")
	helm_list_output, helm_list_err := utils.RunCommandOnContainer("helm", "list", "--all-namespaces")
	if helm_list_err != nil {
		return helm_list_err
	}

	helm_list_err = utils.WriteToFile(helmListFile, helm_list_output)
	if helm_list_err != nil {
		return helm_list_err
	}

	collector.AddToCollectorFiles(helmListFile)

	helmHistoryFile := filepath.Join(rootPath, "helm_history")
	helm_history_output, helm_history_err := utils.RunCommandOnContainer("helm", "history", "-n", "default", "azure-arc")
	if helm_history_err != nil {
		return helm_history_err
	}

	helm_history_err = utils.WriteToFile(helmHistoryFile, helm_history_output)
	if helm_history_err != nil {
		return helm_history_err
	}

	collector.AddToCollectorFiles(helmHistoryFile)

	return nil
}
