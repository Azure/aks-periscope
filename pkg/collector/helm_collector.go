package collector

import (
<<<<<<< HEAD
	"path/filepath"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
=======
	"github.com/Azure/aks-periscope/pkg/interfaces"
>>>>>>> d180a5d (remove secrets)
)

// HelmCollector defines a Helm Collector struct
type HelmCollector struct {
	BaseCollector
}

<<<<<<< HEAD
var _ interfaces.Collector = &HelmCollector{}
=======
var _ interfaces.Collector = &IPTablesCollector{}
>>>>>>> d180a5d (remove secrets)

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
<<<<<<< HEAD
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	helmListFile := filepath.Join(rootPath, "helm_list")
	output, err := utils.RunCommandOnContainer("helm", "list", "--all-namespaces")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(helmListFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(helmListFile)

	helmHistoryFile := filepath.Join(rootPath, collector.GetName())
	output, err = utils.RunCommandOnContainer("helm", "history", "-n", "default", "azure-arc")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(helmHistoryFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(helmHistoryFile)
=======
>>>>>>> d180a5d (remove secrets)
	return nil
}
