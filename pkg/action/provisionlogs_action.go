package action

import (
	"path/filepath"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type provisionLogsAction struct {
	name                     string
	collectIntervalInSeconds int
	processIntervalInSeconds int
	exportIntervalInSeconds  int
	exporter                 interfaces.Exporter
}

var _ interfaces.Action = &provisionLogsAction{}

// NewProvisionLogsAction is a constructor
func NewProvisionLogsAction(collectIntervalInSeconds int, processIntervalInSeconds int, exportIntervalInSeconds int, exporter interfaces.Exporter) interfaces.Action {
	return &provisionLogsAction{
		name:                     "provisionlogs",
		collectIntervalInSeconds: collectIntervalInSeconds,
		processIntervalInSeconds: processIntervalInSeconds,
		exportIntervalInSeconds:  exportIntervalInSeconds,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *provisionLogsAction) GetName() string {
	return action.name
}

// Collect implements the interface method
func (action *provisionLogsAction) Collect() ([]string, error) {
	rootPath, _ := utils.CreateCollectorDir(action.GetName())
	provisionLog := filepath.Join(rootPath, action.GetName())

	go func(provisionLog string) {
		ticker := time.NewTicker(time.Duration(action.collectIntervalInSeconds) * time.Second)
		for ; true; <-ticker.C {
			collectProvisionLogs(provisionLog)
		}
	}(provisionLog)

	return []string{provisionLog}, nil
}

// Process implements the interface method
func (action *provisionLogsAction) Process(collectFiles []string) ([]string, error) {
	return nil, nil
}

// Export implements the interface method
func (action *provisionLogsAction) Export(exporter interfaces.Exporter, collectFiles []string, processfiles []string) error {
	if exporter != nil {
		return exporter.Export(append(collectFiles, processfiles...), action.exportIntervalInSeconds)
	}

	return nil
}

func collectProvisionLogs(provisionLog string) error {
	output, _ := utils.RunCommandOnHost("cat", "/var/log/azure/cluster-provision.log")
	err := utils.WriteToFile(provisionLog, output)
	if err != nil {
		return err
	}

	return nil
}
