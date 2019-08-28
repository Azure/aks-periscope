package action

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type iptablesAction struct {
	name                     string
	collectIntervalInSeconds int
	processIntervalInSeconds int
	exportIntervalInSeconds  int
	exporter                 interfaces.Exporter
}

var _ interfaces.Action = &iptablesAction{}

// NewIPTablesAction is a constructor
func NewIPTablesAction(collectIntervalInSeconds int, processIntervalInSeconds int, exportIntervalInSeconds int, exporter interfaces.Exporter) interfaces.Action {
	return &iptablesAction{
		name:                     "iptables",
		collectIntervalInSeconds: collectIntervalInSeconds,
		processIntervalInSeconds: processIntervalInSeconds,
		exportIntervalInSeconds:  exportIntervalInSeconds,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *iptablesAction) GetName() string {
	return action.name
}

// Collect implements the interface method
func (action *iptablesAction) Collect() ([]string, error) {
	rootPath, _ := utils.CreateCollectorDir(action.GetName())

	iptablesFile := filepath.Join(rootPath, action.GetName())
	file, _ := os.Create(iptablesFile)
	defer file.Close()

	output, _ := utils.RunCommandOnHost("iptables", "-t", "nat", "-L")
	_, err := file.Write([]byte(output))
	if err != nil {
		log.Println("Error while dumping iptables: ", err)
	}

	return []string{iptablesFile}, nil
}

// Process implements the interface method
func (action *iptablesAction) Process(collectFiles []string) ([]string, error) {
	return nil, nil
}

// Export implements the interface method
func (action *iptablesAction) Export(exporter interfaces.Exporter, collectFiles []string, processfiles []string) error {
	if exporter != nil {
		return exporter.Export(append(collectFiles, processfiles...), action.exportIntervalInSeconds)
	}

	return nil
}
