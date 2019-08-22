package collector

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// CollectIPTables print out the host's NAT IP Table
func CollectIPTables(name string) ([]string, error) {
	rootPath, _ := utils.CreateCollectorDir(name)

	iptablesFile := filepath.Join(rootPath, name)
	file, _ := os.Create(iptablesFile)
	defer file.Close()

	output, _ := utils.RunCommandOnHost("iptables", "-t", "nat", "-L")
	_, err := file.Write([]byte(output))
	if err != nil {
		log.Println("Error while dumping iptables: ", err)
	}

	return []string{iptablesFile}, nil
}
