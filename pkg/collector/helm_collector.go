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

var _ interfaces.Collector = &IPTablesCollector{}

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
	output, err := utils.RunCommandOnContainer("helm", "list", "--all-namespaces")
	if err != nil {
		return err
	}
	err = utils.WriteToFile(helmListFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(helmListFile)

	output, err = utils.RunCommandOnContainer("helm", "repo", "add", "kured", "https://weaveworks.github.io/kured/")
	if err != nil {
		return err
	}
	testLog := filepath.Join(rootPath, "helm_repos")
	output, err = utils.RunCommandOnContainer("helm", "search", "repo")
	if err != nil {
		return err
	}
	err = utils.WriteToFile(testLog, output)
	if err != nil {
		return err
	}
	collector.AddToCollectorFiles(testLog)
	output, err = utils.RunCommandOnContainer("helm", "upgrade", "--install", "azure-arc", "kured/kured")
	if err != nil {
		return err
	}
	helmHistoryFile := filepath.Join(rootPath, collector.GetName())
	output, err = utils.RunCommandOnContainer("helm", "history", "azure-arc")
	if err != nil {
		return err
	}
	err = utils.WriteToFile(helmHistoryFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(helmHistoryFile)
	return nil
}
