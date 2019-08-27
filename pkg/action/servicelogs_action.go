package action

import (
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// ServiceLogsAction defines an action on service logs
type ServiceLogsAction struct{}

var _ interfaces.Action = &ServiceLogsAction{}

// GetName implements the interface method
func (action *ServiceLogsAction) GetName() string {
	return "servicelogs"
}

// Collect implements the interface method
func (action *ServiceLogsAction) Collect() ([]string, error) {
	services := []string{"docker", "kubelet"}

	systemLogs := make([]string, 0)

	rootPath, _ := utils.CreateCollectorDir(action.GetName())

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

// Process implements the interface method
func (action *ServiceLogsAction) Process([]string) error {
	return nil
}
