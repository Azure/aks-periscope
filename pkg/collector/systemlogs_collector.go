package collector

import (
	"fmt"
	"runtime"

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

func (collector *SystemLogsCollector) CheckSupported() error {
	// This uses `journalctl` to retrieve system logs, which is not available on Windows.
	// It may be possible in future to identify useful Windows log files and configure this to
	// output those.
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
	}

	return nil
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
