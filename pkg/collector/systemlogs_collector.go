package collector

import (
	"github.com/Azure/aks-periscope/pkg/utils"
)

// SystemLogsCollector defines a SystemLogs Collector struct
type SystemLogsCollector struct {
	data map[string]string
}

// NewSystemLogsCollector is a constructor
func NewSystemLogsCollector() *SystemLogsCollector {
	return &SystemLogsCollector{
		data: make(map[string]string),
	}
}

func (collector *SystemLogsCollector) GetName() string {
	return "systemlogs"
}

// Collect implements the interface method
func (collector *SystemLogsCollector) Collect() error {
	systemServices := []string{"docker", "kubelet"}

	for _, systemService := range systemServices {
		output, err := utils.RunCommandOnHost("journalctl", "-u", systemService)
		if err != nil {
			return err
		}

		collector.data[systemService] = output
	}

	return nil
}

func (collector *SystemLogsCollector) GetData() map[string]string {
	return collector.data
}
