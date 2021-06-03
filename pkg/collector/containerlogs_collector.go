package collector

import (
	"os"
	"strings"

	"github.com/Azure/aks-periscope/pkg/utils"
)

// ContainerLogsCollector defines a ContainerLogs Collector struct
type ContainerLogsCollector struct {
	data map[string]string
}

// NewContainerLogsCollector is a constructor
func NewContainerLogsCollector() *ContainerLogsCollector {
	return &ContainerLogsCollector{
		data: make(map[string]string),
	}
}

func (collector *ContainerLogsCollector) GetName() string {
	return "containerlogs"
}

// Collect implements the interface method
func (collector *ContainerLogsCollector) Collect() error {
	containerLogs := strings.Fields(os.Getenv("DIAGNOSTIC_CONTAINERLOGS_LIST"))

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
			output, err := utils.RunCommandOnHost("docker", "logs", containerName)
			if err != nil {
				return err
			}

			collector.data[containerName] = output
		}
	}

	return nil
}

func (collector *ContainerLogsCollector) GetData() map[string]string {
	return collector.data
}
