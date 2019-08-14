package actions

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// ProvisionLogs collects node provision logs
func ProvisionLogs() (string, error) {
	rootPath := filepath.Join("/aks-diagnostic", utils.GetHostName())
	err := os.MkdirAll(rootPath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	provisionlogsFile := filepath.Join(rootPath, "provisionlogs")
	file, _ := os.Create(provisionlogsFile)
	defer file.Close()

	output, _ := utils.RunCommandOnHost("cat", "/var/log/azure/cluster-provision.log")
	_, err = file.Write([]byte(output))

	if err != nil {
		log.Println("Error getting /var/log/azure/cluster-provision.log: ", err)
	}

	return provisionlogsFile, nil
}
