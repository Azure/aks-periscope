package datacollector

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// Snapshot dumps a snapshot of node info
func Snapshot() (string, error) {
	rootPath := filepath.Join("/aks-diagnostic", utils.GetHostName())
	err := os.MkdirAll(rootPath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	snapshotFile := filepath.Join(rootPath, "snapshot")
	file, _ := os.Create(snapshotFile)
	defer file.Close()

	output, _ := utils.RunCommandOnHost("cat", "/etc/resolv.conf")
	_, err = file.Write([]byte(output))

	if err != nil {
		log.Println("Error while taking snapshot of /etc/resolv.conf: ", err)
	}

	return snapshotFile, nil
}
