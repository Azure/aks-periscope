package processor

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// DNSDiagnosticData defines DNS diagnostic data
type DNSDiagnosticData struct {
	LeveL       string   `json:"Level"`
	NameServers []string `json:"NameServer"`
	Custom      bool     `json:"Custom"`
}

// ProcessDNS processes DNS metrics
func ProcessDNS(name string, files []string) error {
	rootPath, _ := utils.CreateDiagnosticDir()
	DNSDiagnosticFile := filepath.Join(rootPath, name)

	ticker := time.NewTicker(time.Duration(ProcessIntervalInSeconds) * time.Second)
	for {
		select {
		case <-ticker.C:
			processDNS(files, DNSDiagnosticFile)
		}
	}
}

func processDNS(files []string, DNSDiagnosticFile string) error {
	f, _ := os.OpenFile(DNSDiagnosticFile, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	var dnsDiagnosticData []DNSDiagnosticData

	for _, file := range files {
		t, _ := os.Open(file)
		defer t.Close()

		dnsLevel := filepath.Base(file)
		var dns []string
		var search []string
		var isCustom bool
		scanner := bufio.NewScanner(t)
		for scanner.Scan() {
			s := strings.Split(scanner.Text(), " ")
			if s[0] == "nameserver" {
				dns = append(dns, strings.TrimSuffix(s[1], "\n"))
			}
			if s[0] == "search" {
				search = append(search, strings.TrimSuffix(s[1], "\n"))
			}
		}

		if dnsLevel == "host" {
			isCustom = strings.HasPrefix(search[0], "reddog.microsoft.com")
		}

		if dnsLevel == "container" {
			isCustom = dns[0] != "10.0.0.10"
		}

		dataPoint := DNSDiagnosticData{
			LeveL:       dnsLevel,
			NameServers: dns,
			Custom:      isCustom,
		}
		dnsDiagnosticData = append(dnsDiagnosticData, dataPoint)

	}

	for _, dataPoint := range dnsDiagnosticData {
		dataBytes, _ := json.Marshal(dataPoint)
		f.WriteString(string(dataBytes) + "\n")
	}

	return nil
}
