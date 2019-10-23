package action

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
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

	containerLogs := strings.Fields(os.Getenv("DIAGNOSTIC_CONTAINERLOGS_LIST"))
	rootPath, err := utils.CreateCollectorDir(action.GetName())
	if err != nil {
		return err
	}

	output, err := utils.RunCommandOnHost("docker", "ps", "--format", "{{.Names}}")
	if err != nil {
		return err
	}

	containers := strings.Split(output, "\n")
	containers = containers[:len(containers)-1]

	for _, containerLog := range containerLogs {
		containerLogParts := strings.Split(containerLog, "/")
		nameSpace := containerLogParts[0]
		containerNames := []string{}

		for _, container := range containers {
			parts := strings.Split(container, "_")
			if len(containerLogParts) == 2 {
				if parts[1] != "POD" && strings.HasPrefix(parts[2], containerLogParts[1]) && parts[3] == nameSpace {
					containerNames = append(containerNames, strings.TrimPrefix(container, "/"))
				}
			} else {
				if parts[1] != "POD" && parts[3] == nameSpace {
					containerNames = append(containerNames, strings.TrimPrefix(container, "/"))
				}
			}
		}

		for _, containerName := range containerNames {
			parts := strings.Split(containerName, "_")
			containerLog := filepath.Join(rootPath, nameSpace+"_"+parts[2])

			output, err := utils.RunCommandOnHost("docker", "logs", containerName)
			if err != nil {
				return err
			}

			err = utils.WriteToFile(containerLog, output)
			if err != nil {
				return err
			}

			action.collectFiles = append(action.collectFiles, containerLog)
		}
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
