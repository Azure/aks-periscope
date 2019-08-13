package actions

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// DumpIPTables check network connectivity
func DumpIPTables() (string, error) {
	rootPath := filepath.Join("/aks-diagnostic", utils.GetHostName())
	err := os.MkdirAll(rootPath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	iptablesFile := filepath.Join(rootPath, "iptables")
	file, _ := os.Create(iptablesFile)
	defer file.Close()

	output, _ := utils.RunCommandOnHost("iptables", "-t", "nat", "-L")
	_, err = file.Write([]byte(output))

	if err != nil {
		log.Println("Error while dumping iptables: ", err)
	}

	return iptablesFile, nil
}
