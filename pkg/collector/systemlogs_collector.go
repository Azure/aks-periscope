package collector

import (
	"fmt"
	"strings"

	"github.com/Azure/aks-periscope/pkg/utils"
)

// SystemLogsCollector defines a SystemLogs Collector struct
type SystemLogsCollector struct {
	data        map[string]string
	runtimeInfo *utils.RuntimeInfo
}

// NewSystemLogsCollector is a constructor
func NewSystemLogsCollector(runtimeInfo *utils.RuntimeInfo) *SystemLogsCollector {
	return &SystemLogsCollector{
		data:        make(map[string]string),
		runtimeInfo: runtimeInfo,
	}
}

func (collector *SystemLogsCollector) GetName() string {
	return "systemlogs"
}

func (collector *SystemLogsCollector) CheckSupported() error {
	// This uses `journalctl` to retrieve system logs, which is not available on Windows.
	// It may be possible in future to identify useful Windows log files and configure this to
	// output those.
	if collector.runtimeInfo.OSIdentifier != "linux" {
		return fmt.Errorf("Unsupported OS: %s", collector.runtimeInfo.OSIdentifier)
	}

	if utils.Contains(collector.runtimeInfo.CollectorList, "connectedCluster") {
		return fmt.Errorf("Not included because 'connectedCluster' is in COLLECTOR_LIST variable. Included values: %s", strings.Join(collector.runtimeInfo.CollectorList, " "))
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
