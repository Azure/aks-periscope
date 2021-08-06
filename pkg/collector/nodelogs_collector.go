package collector

import (
	"os"
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

// Collect implements the interface method
func (collector *NodeLogsCollector) Collect() error {
	nodeLogs := strings.Fields(os.Getenv("DIAGNOSTIC_NODELOGS_LIST"))

	for _, nodeLog := range nodeLogs {

		output, err := utils.ReadFileContent(nodeLog)
		if err != nil {
			return err
		}

		collector.data["nodeLog"] = output
	}

	return nil
}

func (collector *NodeLogsCollector) GetData() map[string]string {
	return collector.data
}
