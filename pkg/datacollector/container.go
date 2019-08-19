package datacollector

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// PollContainerLogs poll container logs in namespace
func PollContainerLogs(podNameSpace string) ([]string, error) {
	containerNames := make([]string, 0)
	containerLogs := make([]string, 0)

	rootPath := filepath.Join("/aks-diagnostic", utils.GetHostName())
	err := os.MkdirAll(rootPath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	output, _ := utils.RunCommandOnHost("docker", "ps", "--format", "{{.Names}}")
	containers := strings.Split(output, "\n")
	containers = containers[:len(containers)-1]

	for _, container := range containers {
		parts := strings.Split(container, "_")
		if parts[1] != "POD" && parts[3] == podNameSpace {
			containerNames = append(containerNames, strings.TrimPrefix(container, "/"))
		}
	}

	for _, containerName := range containerNames {
		output, _ := utils.RunCommandOnHost("docker", "logs", containerName)

		containerLog := filepath.Join(rootPath, containerName)
		file, _ := os.Create(containerLog)
		defer file.Close()

		_, err = file.Write([]byte(output))

		containerLogs = append(containerLogs, containerLog)
	}

	return containerLogs, nil
}
