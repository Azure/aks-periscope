package collector

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// ContainerLogsCollector defines a ContainerLogs Collector struct
type ContainerLogsCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &ContainerLogsCollector{}

// NewContainerLogsCollector is a constructor
func NewContainerLogsCollector(exporter interfaces.Exporter) *ContainerLogsCollector {
	return &ContainerLogsCollector{
		BaseCollector: BaseCollector{
			collectorType: ContainerLogs,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *ContainerLogsCollector) Collect() error {
	containerLogs := strings.Fields(os.Getenv("DIAGNOSTIC_CONTAINERLOGS_LIST"))
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
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

			collector.AddToCollectorFiles(containerLog)
		}
	}

	return nil
}
