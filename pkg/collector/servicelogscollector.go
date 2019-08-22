package collector

import (
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// CollectServiceLogs collect systemd service logs
func CollectServiceLogs(name string) ([]string, error) {
	services := []string{"docker", "kubelet"}

	systemLogs := make([]string, 0)

	rootPath, _ := utils.CreateCollectorDir(name)

	for _, service := range services {
		output, _ := utils.RunCommandOnHost("journalctl", "-u", service)

		systemLog := filepath.Join(rootPath, service)
		file, _ := os.Create(systemLog)
		defer file.Close()

		file.Write([]byte(output))

		systemLogs = append(systemLogs, systemLog)
	}

	return systemLogs, nil
}
