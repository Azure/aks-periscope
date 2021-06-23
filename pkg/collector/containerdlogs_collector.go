package collector

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// ContainerDLogsCollector defines a ContainerDLogs Collector struct
type ContainerDLogsCollector struct {
	BaseCollector
}

// ContainersList defines configuration for list of containers and their details
type ContainersList struct {
	ContainersList []ContainerInformation `json:"containers"`
}

// ContainerInformation defines configuration of information available for each container
type ContainerInformation struct {
	ID     string            `json:"id"`
	Labels map[string]string `json:"labels"`
}

var _ interfaces.Collector = &ContainerDLogsCollector{}

// NewContainerDLogsCollector is a constructor
func NewContainerDLogsCollector(exporter interfaces.Exporter) *ContainerDLogsCollector {
	return &ContainerDLogsCollector{
		BaseCollector: BaseCollector{
			collectorType: ContainerDLogs,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *ContainerDLogsCollector) Collect() error {
	containerLogs := strings.Fields(os.Getenv("DIAGNOSTIC_CONTAINERLOGS_LIST"))
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	output, err := utils.RunCommandOnHost("crictl", "ps", "-a", "-o", "json")
	if err != nil {
		return err
	}

	var containers ContainersList
	if err = json.Unmarshal([]byte(output), &containers); err != nil {
		log.Fatalf("unable to read containers json file: %v", err)
	}

	for _, containerLog := range containerLogs {
		containerLogParts := strings.Split(containerLog, "/")
		targetNameSpace := containerLogParts[0]
		containerNames := map[string]string{}

		for _, container := range containers.ContainersList {
			containerID := container.ID
			containerNameSpace, ok := container.Labels["io.kubernetes.pod.namespace"]
			if !ok {
				log.Printf("Unable to retrieve namespace of container: %+v", containerID)
				continue
			}

			containerName, ok := container.Labels["io.kubernetes.container.name"]
			if !ok {
				log.Printf("Unable to retrieve name of container: %+v", containerID)
				continue
			}

			if len(containerLogParts) == 2 {
				if strings.HasPrefix(containerName, containerLogParts[1]) && containerNameSpace == targetNameSpace {
					containerNames[containerID] = containerName
				}
			} else {
				if containerNameSpace == targetNameSpace {
					containerNames[containerID] = containerName
				}
			}
		}

		for containerID, containerName := range containerNames {
			containerLog := filepath.Join(rootPath, targetNameSpace+"_"+containerName)
			output, err := utils.RunCommandOnHost("crictl", "logs", containerID)
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
