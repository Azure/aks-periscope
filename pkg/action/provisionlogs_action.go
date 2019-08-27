package action

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// ProvisionLogsAction defines an action on provision logs
type ProvisionLogsAction struct{}

var _ interfaces.Action = &ProvisionLogsAction{}

// GetName implements the interface method
func (action *ProvisionLogsAction) GetName() string {
	return "provisionlogs"
}

// Collect implements the interface method
func (action *ProvisionLogsAction) Collect() ([]string, error) {
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
func (action *ProvisionLogsAction) Process([]string) error {
	return nil
}
