package action

import (
	"path/filepath"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

type systemPerfAction struct {
	name                     string
	collectIntervalInSeconds int
	collectCountForProcess   int
	collectCountForExport    int
	exporter                 interfaces.Exporter
	collectFiles             []string
	processFiles             []string
}

var _ interfaces.Action = &systemPerfAction{}

// NewSystemPerfAction is a constructor
func NewSystemPerfAction(collectIntervalInSeconds int, collectCountForProcess int, collectCountForExport int, exporter interfaces.Exporter) interfaces.Action {
	return &systemPerfAction{
		name:                     "systemperf",
		collectIntervalInSeconds: collectIntervalInSeconds,
		collectCountForProcess:   collectCountForProcess,
		collectCountForExport:    collectCountForExport,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *systemPerfAction) GetName() string {
	return action.name
}

// GetName implements the interface method
func (action *systemPerfAction) GetCollectIntervalInSeconds() int {
	return action.collectIntervalInSeconds
}

// GetName implements the interface method
func (action *systemPerfAction) GetCollectCountForProcess() int {
	return action.collectCountForProcess
}

// GetName implements the interface method
func (action *systemPerfAction) GetCollectCountForExport() int {
	return action.collectCountForExport
}

// Collect implements the interface method
func (action *systemPerfAction) Collect() error {
	action.collectFiles = []string{}

	rootPath, err := utils.CreateCollectorDir(action.GetName())
	if err != nil {
		return err
	}

	topNodesFile := filepath.Join(rootPath, "nodes")
	topPodsFile := filepath.Join(rootPath, "pods")

	output, err := utils.RunCommandOnHost("kubectl", "--kubeconfig", "/var/lib/kubelet/kubeconfig", "top", "nodes")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(topNodesFile, output)
	if err != nil {
		return err
	}

	action.collectFiles = append(action.collectFiles, topNodesFile)

	output, err = utils.RunCommandOnHost("kubectl", "--kubeconfig", "/var/lib/kubelet/kubeconfig", "top", "pods", "--all-namespaces")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(topPodsFile, output)
	if err != nil {
		return err
	}

	action.collectFiles = append(action.collectFiles, topPodsFile)

	return nil
}

// Process implements the interface method
func (action *systemPerfAction) Process() error {
	return nil
}

// Export implements the interface method
func (action *systemPerfAction) Export() error {
	if action.exporter != nil {
		return action.exporter.Export(append(action.collectFiles, action.processFiles...))
	}

	return nil
}
