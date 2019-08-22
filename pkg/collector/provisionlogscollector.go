package collector

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// CollectProvisionLogs collects node provision logs
func CollectProvisionLogs(name string) ([]string, error) {
	rootPath, _ := utils.CreateCollectorDir(name)

	provisionlogsFile := filepath.Join(rootPath, name)
	file, _ := os.Create(provisionlogsFile)
	defer file.Close()

	output, _ := utils.RunCommandOnHost("cat", "/var/log/azure/cluster-provision.log")
	_, err := file.Write([]byte(output))
	if err != nil {
		log.Println("Error getting /var/log/azure/cluster-provision.log: ", err)
	}

	return []string{provisionlogsFile}, nil
}
