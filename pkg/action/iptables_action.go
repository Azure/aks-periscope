package action

import (
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type iptablesAction struct {
	name                     string
	collectIntervalInSeconds int
	collectCountForProcess   int
	collectCountForExport    int
	exporter                 interfaces.Exporter
	collectFiles             []string
	processFiles             []string
}

var _ interfaces.Action = &iptablesAction{}

// NewIPTablesAction is a constructor
func NewIPTablesAction(collectIntervalInSeconds int, collectCountForProcess int, collectCountForExport int, exporter interfaces.Exporter) interfaces.Action {
	return &iptablesAction{
		name:                     "iptables",
		collectIntervalInSeconds: collectIntervalInSeconds,
		collectCountForProcess:   collectCountForProcess,
		collectCountForExport:    collectCountForExport,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *iptablesAction) GetName() string {
	return action.name
}

// GetName implements the interface method
func (action *iptablesAction) GetCollectIntervalInSeconds() int {
	return action.collectIntervalInSeconds
}

// GetName implements the interface method
func (action *iptablesAction) GetCollectCountForProcess() int {
	return action.collectCountForProcess
}

// GetName implements the interface method
func (action *iptablesAction) GetCollectCountForExport() int {
	return action.collectCountForExport
}

// Collect implements the interface method
func (action *iptablesAction) Collect() error {
	action.collectFiles = []string{}

	rootPath, _ := utils.CreateCollectorDir(action.GetName())
	iptablesFile := filepath.Join(rootPath, action.GetName())

	output, _ := utils.RunCommandOnHost("iptables", "-t", "nat", "-L")
	err := utils.WriteToFile(iptablesFile, output)
	if err != nil {
		return err
	}

	action.collectFiles = append(action.collectFiles, iptablesFile)

	return nil
}

// Process implements the interface method
func (action *iptablesAction) Process() error {
	return nil
}

// Export implements the interface method
func (action *iptablesAction) Export() error {
	if action.exporter != nil {
		return action.exporter.Export(append(action.collectFiles, action.processFiles...))
	}

	return nil
}
