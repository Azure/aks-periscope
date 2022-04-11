package collector

import (
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// NodeLogsCollector defines a NodeLogs Collector struct
type NodeLogsCollector struct {
	data        map[string]string
	runtimeInfo *utils.RuntimeInfo
	fileReader  interfaces.FileContentReader
}

// NewNodeLogsCollector is a constructor
func NewNodeLogsCollector(runtimeInfo *utils.RuntimeInfo, fileReader interfaces.FileContentReader) *NodeLogsCollector {
	return &NodeLogsCollector{
		data:        make(map[string]string),
		runtimeInfo: runtimeInfo,
		fileReader:  fileReader,
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
	for _, nodeLog := range collector.runtimeInfo.NodeLogs {
		normalizedNodeLog := strings.Replace(nodeLog, "/", "_", -1)
		if normalizedNodeLog[0] == '_' {
			normalizedNodeLog = normalizedNodeLog[1:]
		}

		output, err := collector.fileReader.GetFileContent(nodeLog)
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
