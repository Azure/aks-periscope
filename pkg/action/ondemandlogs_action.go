package action

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

type onDemandLogsAction struct {
	name                     string
	collectIntervalInSeconds int
	collectCountForProcess   int
	collectCountForExport    int
	exporter                 interfaces.Exporter
	collectFiles             []string
	processFiles             []string
}

var _ interfaces.Action = &onDemandLogsAction{}

// NewOnDemandLogsAction is a constructor
func NewOnDemandLogsAction(collectIntervalInSeconds int, collectCountForProcess int, collectCountForExport int, exporter interfaces.Exporter) interfaces.Action {
	return &onDemandLogsAction{
		name:                     "ondemandlogs",
		collectIntervalInSeconds: collectIntervalInSeconds,
		collectCountForProcess:   collectCountForProcess,
		collectCountForExport:    collectCountForExport,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *onDemandLogsAction) GetName() string {
	return action.name
}

// GetName implements the interface method
func (action *onDemandLogsAction) GetCollectIntervalInSeconds() int {
	return action.collectIntervalInSeconds
}

// GetName implements the interface method
func (action *onDemandLogsAction) GetCollectCountForProcess() int {
	return action.collectCountForProcess
}

// GetName implements the interface method
func (action *onDemandLogsAction) GetCollectCountForExport() int {
	return action.collectCountForExport
}

// Collect implements the interface method
func (action *onDemandLogsAction) Collect() error {
	action.collectFiles = []string{}

	onDemandLogs := strings.Fields(os.Getenv("DIAGNOSTIC_ONDEMANDLOGS_LIST"))
	rootPath, err := utils.CreateCollectorDir(action.GetName())
	if err != nil {
		return err
	}

	for _, onDemandLog := range onDemandLogs {
		normalizedOnDemandLog := strings.Replace(onDemandLog, "/", "-", -1)
		if normalizedOnDemandLog[0] == '-' {
			normalizedOnDemandLog = normalizedOnDemandLog[1:]
		}

		onDemandLogFile := filepath.Join(rootPath, normalizedOnDemandLog)

		output, err := utils.RunCommandOnHost("cat", onDemandLog)
		if err != nil {
			return err
		}

		err = utils.WriteToFile(onDemandLogFile, output)
		if err != nil {
			return err
		}

		action.collectFiles = append(action.collectFiles, onDemandLogFile)
	}

	return nil
}

// Process implements the interface method
func (action *onDemandLogsAction) Process() error {
	return nil
}

// Export implements the interface method
func (action *onDemandLogsAction) Export() error {
	if action.exporter != nil {
		return action.exporter.Export(append(action.collectFiles, action.processFiles...))
	}

	return nil
}
