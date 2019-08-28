package action

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type provisionLogsAction struct {
	name                     string
	collectIntervalInSeconds int
	processIntervalInSeconds int
	exportIntervalInSeconds  int
	exporter                 interfaces.Exporter
}

var _ interfaces.Action = &provisionLogsAction{}

// NewProvisionLogsAction is a constructor
func NewProvisionLogsAction(collectIntervalInSeconds int, processIntervalInSeconds int, exportIntervalInSeconds int, exporter interfaces.Exporter) interfaces.Action {
	return &provisionLogsAction{
		name:                     "provisionlogs",
		collectIntervalInSeconds: collectIntervalInSeconds,
		processIntervalInSeconds: processIntervalInSeconds,
		exportIntervalInSeconds:  exportIntervalInSeconds,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *provisionLogsAction) GetName() string {
	return action.name
}

// Collect implements the interface method
func (action *provisionLogsAction) Collect() ([]string, error) {
	rootPath, _ := utils.CreateCollectorDir(action.GetName())

	provisionlogsFile := filepath.Join(rootPath, action.GetName())
	file, _ := os.Create(provisionlogsFile)
	defer file.Close()

	output, _ := utils.RunCommandOnHost("cat", "/var/log/azure/cluster-provision.log")
	_, err := file.Write([]byte(output))
	if err != nil {
		log.Println("Error getting /var/log/azure/cluster-provision.log: ", err)
	}

	return []string{provisionlogsFile}, nil
}

// Process implements the interface method
func (action *provisionLogsAction) Process(collectFiles []string) ([]string, error) {
	return nil, nil
}

// Export implements the interface method
func (action *provisionLogsAction) Export(exporter interfaces.Exporter, collectFiles []string, processfiles []string) error {
	if exporter != nil {
		return exporter.Export(append(collectFiles, processfiles...), action.exportIntervalInSeconds)
	}

	return nil
}
