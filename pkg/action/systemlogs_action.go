package action

import (
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type systemLogsAction struct {
	name                     string
	collectIntervalInSeconds int
	collectCountForProcess   int
	collectCountForExport    int
	exporter                 interfaces.Exporter
	collectFiles             []string
	processFiles             []string
}

var _ interfaces.Action = &systemLogsAction{}

// NewSystemLogsAction is a constructor
func NewSystemLogsAction(collectIntervalInSeconds int, collectCountForProcess int, collectCountForExport int, exporter interfaces.Exporter) interfaces.Action {
	return &systemLogsAction{
		name:                     "systemlogs",
		collectIntervalInSeconds: collectIntervalInSeconds,
		collectCountForProcess:   collectCountForProcess,
		collectCountForExport:    collectCountForExport,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *systemLogsAction) GetName() string {
	return action.name
}

// GetName implements the interface method
func (action *systemLogsAction) GetCollectIntervalInSeconds() int {
	return action.collectIntervalInSeconds
}

// GetName implements the interface method
func (action *systemLogsAction) GetCollectCountForProcess() int {
	return action.collectCountForProcess
}

// GetName implements the interface method
func (action *systemLogsAction) GetCollectCountForExport() int {
	return action.collectCountForExport
}

// Collect implements the interface method
func (action *systemLogsAction) Collect() error {
	action.collectFiles = []string{}

	systemServices := []string{"docker", "kubelet"}
	rootPath, err := utils.CreateCollectorDir(action.GetName())
	if err != nil {
		return err
	}

	for _, systemService := range systemServices {
		systemLog := filepath.Join(rootPath, systemService)

		output, err := utils.RunCommandOnHost("journalctl", "-u", systemService)
		if err != nil {
			return err
		}

		err = utils.WriteToFile(systemLog, output)
		if err != nil {
			return err
		}

		action.collectFiles = append(action.collectFiles, systemLog)
	}

	return nil
}

// Process implements the interface method
func (action *systemLogsAction) Process() error {
	return nil
}

// Export implements the interface method
func (action *systemLogsAction) Export() error {
	if action.exporter != nil {
		return action.exporter.Export(append(action.collectFiles, action.processFiles...))
	}

	return nil
}
