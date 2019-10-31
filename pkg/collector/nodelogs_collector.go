package collector

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// NodeLogsCollector defines a NodeLogs Collector struct
type NodeLogsCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &NodeLogsCollector{}

// NewNodeLogsCollector is a constructor
func NewNodeLogsCollector(exporter interfaces.Exporter) *NodeLogsCollector {
	return &NodeLogsCollector{
		BaseCollector: BaseCollector{
			collectorType: NodeLogs,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *NodeLogsCollector) Collect() error {
	nodeLogs := strings.Fields(os.Getenv("DIAGNOSTIC_NODELOGS_LIST"))
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	for _, nodeLog := range nodeLogs {
		normalizedNodeLog := strings.Replace(nodeLog, "/", "_", -1)
		if normalizedNodeLog[0] == '_' {
			normalizedNodeLog = normalizedNodeLog[1:]
		}

		nodeLogFile := filepath.Join(rootPath, normalizedNodeLog)

		output, err := utils.RunCommandOnHost("cat", nodeLog)
		if err != nil {
			return err
		}

		err = utils.WriteToFile(nodeLogFile, output)
		if err != nil {
			return err
		}

		collector.AddToCollectorFiles(nodeLogFile)
	}

	return nil
}
