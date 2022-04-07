package collector

import (
	"os"
	"runtime"
	"strings"

	"github.com/Azure/aks-periscope/pkg/utils"
)

// NodeLogsCollector defines a NodeLogs Collector struct
type NodeLogsCollector struct {
	data map[string]string
}

// NewNodeLogsCollector is a constructor
func NewNodeLogsCollector() *NodeLogsCollector {
	return &NodeLogsCollector{
		data: make(map[string]string),
	}
}

func (collector *NodeLogsCollector) GetName() string {
	return "nodelogs"
}

func (collector *NodeLogsCollector) CheckSupported() error {
	// Although the files read by this collector may be different between Windows and Linux,
	// they are defined in a ConfigMap which is expected to be populated correctly for the OS.
	return nil
}

// Collect implements the interface method
func (collector *NodeLogsCollector) Collect() error {
	var nodeLogs []string
	if runtime.GOOS == "linux" {
		nodeLogs = strings.Fields(os.Getenv("DIAGNOSTIC_NODELOGS_LIST_LINUX"))
	} else {
		nodeLogs = strings.Fields(os.Getenv("DIAGNOSTIC_NODELOGS_LIST_WINDOWS"))
	}

	for _, nodeLog := range nodeLogs {
		normalizedNodeLog := strings.Replace(nodeLog, "/", "_", -1)
		if normalizedNodeLog[0] == '_' {
			normalizedNodeLog = normalizedNodeLog[1:]
		}

		output, err := utils.ReadFileContent(nodeLog)
		if err != nil {
			return err
		}

		collector.data[normalizedNodeLog] = output
	}

	return nil
}

func (collector *NodeLogsCollector) GetData() map[string]string {
	return collector.data
}
