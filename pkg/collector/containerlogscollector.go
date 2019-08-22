package collector

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// CollectContainerLogs collect container logs in namespace
func CollectContainerLogs(name string) ([]string, error) {
	podNameSpace := "kube-system"

	containerNames := make([]string, 0)
	containerLogs := make([]string, 0)

	rootPath, _ := utils.CreateCollectorDir(name)

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

		file.Write([]byte(output))

		containerLogs = append(containerLogs, containerLog)
	}

	return containerLogs, nil
}
