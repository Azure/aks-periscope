package action

import (
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type kubeletCmdAction struct {
	name                     string
	collectIntervalInSeconds int
	collectCountForProcess   int
	collectCountForExport    int
	exporter                 interfaces.Exporter
	collectFiles             []string
	processFiles             []string
}

var _ interfaces.Action = &kubeletCmdAction{}

// NewKubeletCmdAction is a constructor
func NewKubeletCmdAction(collectIntervalInSeconds int, collectCountForProcess int, collectCountForExport int, exporter interfaces.Exporter) interfaces.Action {
	return &kubeletCmdAction{
		name:                     "kubeletcmd",
		collectIntervalInSeconds: collectIntervalInSeconds,
		collectCountForProcess:   collectCountForProcess,
		collectCountForExport:    collectCountForExport,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *kubeletCmdAction) GetName() string {
	return action.name
}

// GetName implements the interface method
func (action *kubeletCmdAction) GetCollectIntervalInSeconds() int {
	return action.collectIntervalInSeconds
}

// GetName implements the interface method
func (action *kubeletCmdAction) GetCollectCountForProcess() int {
	return action.collectCountForProcess
}

// GetName implements the interface method
func (action *kubeletCmdAction) GetCollectCountForExport() int {
	return action.collectCountForExport
}

// Collect implements the interface method
func (action *kubeletCmdAction) Collect() error {
	action.collectFiles = []string{}

	rootPath, err := utils.CreateCollectorDir(action.GetName())
	if err != nil {
		return err
	}

	kubeletcmdFile := filepath.Join(rootPath, action.GetName())

	output, err := utils.RunCommandOnHost("ps", "-o", "cmd=", "-C", "kubelet")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(kubeletcmdFile, output)
	if err != nil {
		return err
	}

	action.collectFiles = append(action.collectFiles, kubeletcmdFile)

	return nil
}

// Process implements the interface method
func (action *kubeletCmdAction) Process() error {
	return nil
}

// Export implements the interface method
func (action *kubeletCmdAction) Export() error {
	if action.exporter != nil {
		return action.exporter.Export(append(action.collectFiles, action.processFiles...))
	}

	return nil
}
