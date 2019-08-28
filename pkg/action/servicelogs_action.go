package action

import (
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type serviceLogsAction struct {
	name                     string
	collectIntervalInSeconds int
	processIntervalInSeconds int
	exportIntervalInSeconds  int
	exporter                 interfaces.Exporter
}

var _ interfaces.Action = &serviceLogsAction{}

// NewServiceLogsAction is a constructor
func NewServiceLogsAction(collectIntervalInSeconds int, processIntervalInSeconds int, exportIntervalInSeconds int, exporter interfaces.Exporter) interfaces.Action {
	return &serviceLogsAction{
		name:                     "containerlogs",
		collectIntervalInSeconds: collectIntervalInSeconds,
		processIntervalInSeconds: processIntervalInSeconds,
		exportIntervalInSeconds:  exportIntervalInSeconds,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *serviceLogsAction) GetName() string {
	return action.name
}

// Collect implements the interface method
func (action *serviceLogsAction) Collect() ([]string, error) {
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
func (action *serviceLogsAction) Process(collectFiles []string) ([]string, error) {
	return nil, nil
}

// Export implements the interface method
func (action *serviceLogsAction) Export(exporter interfaces.Exporter, collectFiles []string, processfiles []string) error {
	if exporter != nil {
		return exporter.Export(append(collectFiles, processfiles...), action.exportIntervalInSeconds)
	}

	return nil
}
