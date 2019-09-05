package action

import (
	"path/filepath"
	"strings"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type containerLogsAction struct {
	name                     string
	collectIntervalInSeconds int
	collectCountForProcess   int
	collectCountForExport    int
	exporter                 interfaces.Exporter
	collectFiles             []string
	processFiles             []string
}

var _ interfaces.Action = &containerLogsAction{}

// NewContainerLogsAction is a constructor
func NewContainerLogsAction(collectIntervalInSeconds int, collectCountForProcess int, collectCountForExport int, exporter interfaces.Exporter) interfaces.Action {
	return &containerLogsAction{
		name:                     "containerlogs",
		collectIntervalInSeconds: collectIntervalInSeconds,
		collectCountForProcess:   collectCountForProcess,
		collectCountForExport:    collectCountForExport,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *containerLogsAction) GetName() string {
	return action.name
}

// GetName implements the interface method
func (action *containerLogsAction) GetCollectIntervalInSeconds() int {
	return action.collectIntervalInSeconds
}

// GetName implements the interface method
func (action *containerLogsAction) GetCollectCountForProcess() int {
	return action.collectCountForProcess
}

// GetName implements the interface method
func (action *containerLogsAction) GetCollectCountForExport() int {
	return action.collectCountForExport
}

// Collect implements the interface method
func (action *containerLogsAction) Collect() error {
	action.collectFiles = []string{}

	podNameSpace := "kube-system"
	rootPath, _ := utils.CreateCollectorDir(action.GetName())

	output, _ := utils.RunCommandOnHost("docker", "ps", "--format", "{{.Names}}")
	containers := strings.Split(output, "\n")
	containers = containers[:len(containers)-1]

	containerNames := []string{}
	for _, container := range containers {
		parts := strings.Split(container, "_")
		if parts[1] != "POD" && parts[3] == podNameSpace {
			containerNames = append(containerNames, strings.TrimPrefix(container, "/"))
		}
	}

	for _, containerName := range containerNames {
		containerLog := filepath.Join(rootPath, containerName)

		output, _ := utils.RunCommandOnHost("docker", "logs", containerName)
		err := utils.WriteToFile(containerLog, output)
		if err != nil {
			return err
		}

		action.collectFiles = append(action.collectFiles, containerLog)
	}

	return nil
}

// Process implements the interface method
func (action *containerLogsAction) Process() error {
	return nil
}

// Export implements the interface method
func (action *containerLogsAction) Export() error {
	if action.exporter != nil {
		return action.exporter.Export(append(action.collectFiles, action.processFiles...))
	}

	return nil
}
