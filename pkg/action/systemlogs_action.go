package action

import (
	"path/filepath"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type systemLogsAction struct {
	name                     string
	collectIntervalInSeconds int
	processIntervalInSeconds int
	exportIntervalInSeconds  int
	exporter                 interfaces.Exporter
}

var _ interfaces.Action = &systemLogsAction{}

// NewSystemLogsAction is a constructor
func NewSystemLogsAction(collectIntervalInSeconds int, processIntervalInSeconds int, exportIntervalInSeconds int, exporter interfaces.Exporter) interfaces.Action {
	return &systemLogsAction{
		name:                     "systemlogs",
		collectIntervalInSeconds: collectIntervalInSeconds,
		processIntervalInSeconds: processIntervalInSeconds,
		exportIntervalInSeconds:  exportIntervalInSeconds,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *systemLogsAction) GetName() string {
	return action.name
}

// Collect implements the interface method
func (action *systemLogsAction) Collect() ([]string, error) {
	systemServices := []string{"docker", "kubelet"}
	rootPath, _ := utils.CreateCollectorDir(action.GetName())

	systemLogs := []string{}
	for _, systemService := range systemServices {
		systemLog := filepath.Join(rootPath, systemService)

		go func(systemService string, systemLog string) {
			ticker := time.NewTicker(time.Duration(action.collectIntervalInSeconds) * time.Second)
			for {
				select {
				case <-ticker.C:
					collectSystemLogs(systemService, systemLog)
				}
			}
		}(systemService, systemLog)

		systemLogs = append(systemLogs, systemLog)
	}

	return systemLogs, nil
}

// Process implements the interface method
func (action *systemLogsAction) Process(collectFiles []string) ([]string, error) {
	return nil, nil
}

// Export implements the interface method
func (action *systemLogsAction) Export(exporter interfaces.Exporter, collectFiles []string, processfiles []string) error {
	if exporter != nil {
		return exporter.Export(append(collectFiles, processfiles...), action.exportIntervalInSeconds)
	}

	return nil
}

func collectSystemLogs(systemService string, systemLog string) error {
	output, _ := utils.RunCommandOnHost("journalctl", "-u", systemService)
	err := utils.WriteToFile(systemLog, output)
	if err != nil {
		return err
	}

	return nil
}
