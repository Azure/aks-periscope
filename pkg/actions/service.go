package actions

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// PollSystemLogs poll systemd logs using journal client
func PollSystemLogs(services []string) ([]string, error) {
	systemLogs := make([]string, 0)

	rootPath := filepath.Join("/aks-diagnostic", utils.GetHostName())
	err := os.MkdirAll(rootPath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	for _, service := range services {
		output, _ := utils.RunCommandOnHost("journalctl", "-u", service)

		systemLog := filepath.Join(rootPath, service)
		file, _ := os.Create(systemLog)
		defer file.Close()

		_, err = file.Write([]byte(output))

		systemLogs = append(systemLogs, systemLog)
	}

	return systemLogs, nil
}
