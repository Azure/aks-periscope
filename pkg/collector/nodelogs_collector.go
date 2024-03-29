package collector

import (
	"fmt"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// NodeLogsCollector defines a NodeLogs Collector struct
type NodeLogsCollector struct {
	data        map[string]interfaces.DataValue
	runtimeInfo *utils.RuntimeInfo
	fileSystem  interfaces.FileSystemAccessor
}

// NewNodeLogsCollector is a constructor
func NewNodeLogsCollector(runtimeInfo *utils.RuntimeInfo, fileSystem interfaces.FileSystemAccessor) *NodeLogsCollector {
	return &NodeLogsCollector{
		data:        make(map[string]interfaces.DataValue),
		runtimeInfo: runtimeInfo,
		fileSystem:  fileSystem,
	}
}

func (collector *NodeLogsCollector) GetName() string {
	return "nodelogs"
}

func (collector *NodeLogsCollector) CheckSupported() error {
	if utils.Contains(collector.runtimeInfo.CollectorList, "connectedCluster") {
		return fmt.Errorf("not included because 'connectedCluster' is in COLLECTOR_LIST variable. Included values: %s", strings.Join(collector.runtimeInfo.CollectorList, " "))
	}

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

		size, err := collector.fileSystem.GetFileSize(nodeLog)
		if err != nil {
			return fmt.Errorf("error getting file size for %s: %w", nodeLog, err)
		}

		collector.data[normalizedNodeLog] = utils.NewFilePathDataValue(collector.fileSystem, nodeLog, size)
	}

	return nil
}

func (collector *NodeLogsCollector) GetData() map[string]interfaces.DataValue {
	return collector.data
}
