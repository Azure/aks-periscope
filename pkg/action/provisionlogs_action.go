package action

import (
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type provisionLogsAction struct {
	name                     string
	collectIntervalInSeconds int
	collectCountForProcess   int
	collectCountForExport    int
	exporter                 interfaces.Exporter
	collectFiles             []string
	processFiles             []string
}

var _ interfaces.Action = &provisionLogsAction{}

// NewProvisionLogsAction is a constructor
func NewProvisionLogsAction(collectIntervalInSeconds int, collectCountForProcess int, collectCountForExport int, exporter interfaces.Exporter) interfaces.Action {
	return &provisionLogsAction{
		name:                     "provisionlogs",
		collectIntervalInSeconds: collectIntervalInSeconds,
		collectCountForProcess:   collectCountForProcess,
		collectCountForExport:    collectCountForExport,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *provisionLogsAction) GetName() string {
	return action.name
}

// GetName implements the interface method
func (action *provisionLogsAction) GetCollectIntervalInSeconds() int {
	return action.collectIntervalInSeconds
}

// GetName implements the interface method
func (action *provisionLogsAction) GetCollectCountForProcess() int {
	return action.collectCountForProcess
}

// GetName implements the interface method
func (action *provisionLogsAction) GetCollectCountForExport() int {
	return action.collectCountForExport
}

// Collect implements the interface method
func (action *provisionLogsAction) Collect() error {
	action.collectFiles = []string{}

	rootPath, err := utils.CreateCollectorDir(action.GetName())
	if err != nil {
		return err
	}

	provisionLog := filepath.Join(rootPath, action.GetName())

	output, err := utils.RunCommandOnHost("cat", "/var/log/azure/cluster-provision.log")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(provisionLog, output)
	if err != nil {
		return err
	}

	action.collectFiles = append(action.collectFiles, provisionLog)

	return nil
}

// Process implements the interface method
func (action *provisionLogsAction) Process() error {
	return nil
}

// Export implements the interface method
func (action *provisionLogsAction) Export() error {
	if action.exporter != nil {
		return action.exporter.Export(append(action.collectFiles, action.processFiles...))
	}

	return nil
}
