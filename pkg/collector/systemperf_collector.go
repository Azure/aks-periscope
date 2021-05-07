package collector

import (
	"path/filepath"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// SystemPerfCollector defines a SystemPerf Collector struct
type SystemPerfCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &SystemPerfCollector{}

// NewSystemPerfCollector is a constructor
func NewSystemPerfCollector(exporters []interfaces.Exporter) *SystemPerfCollector {
	return &SystemPerfCollector{
		BaseCollector: BaseCollector{
			collectorType: SystemPerf,
			exporters:      exporters,
		},
	}
}

// Collect implements the interface method
func (collector *SystemPerfCollector) Collect() error {
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	err = utils.CreateKubeConfigFromServiceAccount()
	if err != nil {
		return err
	}

	topNodesFile := filepath.Join(rootPath, "nodes")
	topPodsFile := filepath.Join(rootPath, "pods")

	output, err := utils.RunCommandOnContainer("kubectl", "top", "nodes")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(topNodesFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(topNodesFile)

	output, err = utils.RunCommandOnContainer("kubectl", "top", "pods", "--all-namespaces")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(topPodsFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(topPodsFile)

	return nil
}
