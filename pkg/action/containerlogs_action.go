package action

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type containerLogsAction struct {
	name                     string
	collectIntervalInSeconds int
	processIntervalInSeconds int
	exportIntervalInSeconds  int
	exporter                 interfaces.Exporter
}

var _ interfaces.Action = &containerLogsAction{}

// NewContainerLogsAction is a constructor
func NewContainerLogsAction(collectIntervalInSeconds int, processIntervalInSeconds int, exportIntervalInSeconds int, exporter interfaces.Exporter) interfaces.Action {
	return &containerLogsAction{
		name:                     "containerlogs",
		collectIntervalInSeconds: collectIntervalInSeconds,
		processIntervalInSeconds: processIntervalInSeconds,
		exportIntervalInSeconds:  exportIntervalInSeconds,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *containerLogsAction) GetName() string {
	return action.name
}

// Collect implements the interface method
func (action *containerLogsAction) Collect() ([]string, error) {
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

	containerLogs := []string{}
	for _, containerName := range containerNames {
		containerLog := filepath.Join(rootPath, containerName)

		go func(containerName string, containerLog string) {
			ticker := time.NewTicker(time.Duration(action.collectIntervalInSeconds) * time.Second)
			for ; true; <-ticker.C {
				collectContainerLogs(containerName, containerLog)
			}
		}(containerName, containerLog)

		containerLogs = append(containerLogs, containerLog)
	}

	return containerLogs, nil
}

// Process implements the interface method
func (action *containerLogsAction) Process(collectFiles []string) ([]string, error) {
	return nil, nil
}

// Export implements the interface method
func (action *containerLogsAction) Export(exporter interfaces.Exporter, collectFiles []string, processfiles []string) error {
	if exporter != nil {
		return exporter.Export(append(collectFiles, processfiles...), action.exportIntervalInSeconds)
	}

	return nil
}

func collectContainerLogs(containerName string, containerLog string) error {
	output, _ := utils.RunCommandOnHost("docker", "logs", containerName)
	err := utils.WriteToFile(containerLog, output)
	if err != nil {
		return err
	}

	return nil
}
