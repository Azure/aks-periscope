package actions

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
	"github.com/coreos/go-systemd/sdjournal"
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
		journalReaderConfig := sdjournal.JournalReaderConfig{
			Path: "/var/log/journal/",
			Matches: []sdjournal.Match{
				{
					Field: sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT,
					Value: service + ".service",
				},
			},
		}

		jr, err := sdjournal.NewJournalReader(journalReaderConfig)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		systemLog := filepath.Join(rootPath, service)
		file, err := os.Create(systemLog)
		defer file.Close()

		b := make([]byte, 64*1<<(10))
		for {
			c, err := jr.Read(b)
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatal(err)
				return nil, err
			}

			_, err = file.Write(b[:c])
		}

		systemLogs = append(systemLogs, systemLog)
	}

	return systemLogs, nil
}
