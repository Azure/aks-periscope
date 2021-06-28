package collector

import (
	"path/filepath"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// SystemLogsCollector defines a SystemLogs Collector struct
type SystemLogsCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &SystemLogsCollector{}

// NewSystemLogsCollector is a constructor
func NewSystemLogsCollector(exporters []interfaces.Exporter) *SystemLogsCollector {
	return &SystemLogsCollector{
		BaseCollector: BaseCollector{
			collectorType: SystemLogs,
			exporters:     exporters,
		},
	}
}

// Collect implements the interface method
func (collector *SystemLogsCollector) Collect() error {
	systemServices := []string{"docker", "kubelet"}
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	for _, systemService := range systemServices {
		systemLog := filepath.Join(rootPath, systemService)

		output, err := utils.RunCommandOnHost("journalctl", "-u", systemService)
		if err != nil {
			return err
		}

		err = utils.WriteToFile(systemLog, output)
		if err != nil {
			return err
		}

		collector.AddToCollectorFiles(systemLog)
	}

	return nil
}
