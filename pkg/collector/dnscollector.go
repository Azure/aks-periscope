package collector

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// CollectDNS collects host and container level DNS file
func CollectDNS(name string) ([]string, error) {
	rootPath, _ := utils.CreateCollectorDir(name)

	hostDNSFile := filepath.Join(rootPath, "host")
	file, _ := os.Create(hostDNSFile)
	defer file.Close()

	output, _ := utils.RunCommandOnHost("cat", "/etc/resolv.conf")
	_, err := file.Write([]byte(output))
	if err != nil {
		log.Println("Error while taking snapshot of /etc/resolv.conf: ", err)
	}

	containerDNSFile := filepath.Join(rootPath, "container")
	utils.CopyLocalFile("/etc/resolv.conf", containerDNSFile)

	return []string{hostDNSFile, containerDNSFile}, nil
}
