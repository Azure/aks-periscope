package action

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// IPTablesAction defines an action on iptables
type IPTablesAction struct{}

var _ interfaces.Action = &IPTablesAction{}

// GetName implements the interface method
func (action *IPTablesAction) GetName() string {
	return "iptables"
}

// Collect implements the interface method
func (action *IPTablesAction) Collect() ([]string, error) {
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
func (action *IPTablesAction) Process([]string) error {
	return nil
}
