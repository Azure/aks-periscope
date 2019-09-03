package action

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type dnsDiagnosticDatum struct {
	LeveL       string   `json:"Level"`
	NameServers []string `json:"NameServer"`
	Custom      bool     `json:"Custom"`
}

type dnsAction struct {
	name                     string
	collectIntervalInSeconds int
	processIntervalInSeconds int
	exportIntervalInSeconds  int
	exporter                 interfaces.Exporter
}

var _ interfaces.Action = &dnsAction{}

// NewDNSAction is a constructor
func NewDNSAction(collectIntervalInSeconds int, processIntervalInSeconds int, exportIntervalInSeconds int, exporter interfaces.Exporter) interfaces.Action {
	return &dnsAction{
		name:                     "dns",
		collectIntervalInSeconds: collectIntervalInSeconds,
		processIntervalInSeconds: processIntervalInSeconds,
		exportIntervalInSeconds:  exportIntervalInSeconds,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *dnsAction) GetName() string {
	return action.name
}

// Collect implements the interface method
func (action *dnsAction) Collect() ([]string, error) {
	rootPath, _ := utils.CreateCollectorDir(action.GetName())
	hostDNSFile := filepath.Join(rootPath, "host")
	containerDNSFile := filepath.Join(rootPath, "container")

	go func(hostDNSFile string, containerDNSFile string) {
		ticker := time.NewTicker(time.Duration(action.collectIntervalInSeconds) * time.Second)
		for {
			select {
			case <-ticker.C:
				collectDNS(hostDNSFile, containerDNSFile)
			}
		}
	}(hostDNSFile, containerDNSFile)

	return []string{hostDNSFile, containerDNSFile}, nil
}

// Process implements the interface method
func (action *dnsAction) Process(collectFiles []string) ([]string, error) {
	rootPath, _ := utils.CreateDiagnosticDir()
	dnsDiagnosticFile := filepath.Join(rootPath, action.GetName())

	go func(collectFiles []string, dnsDiagnosticFile string) {
		ticker := time.NewTicker(time.Duration(action.processIntervalInSeconds) * time.Second)
		for {
			select {
			case <-ticker.C:
				processDNS(collectFiles, dnsDiagnosticFile)
			}
		}
	}(collectFiles, dnsDiagnosticFile)

	return []string{dnsDiagnosticFile}, nil
}

// Export implements the interface method
func (action *dnsAction) Export(exporter interfaces.Exporter, collectFiles []string, processfiles []string) error {
	if exporter != nil {
		return exporter.Export(append(collectFiles, processfiles...), action.exportIntervalInSeconds)
	}

	return nil
}

func collectDNS(hostDNSFile string, containerDNSFile string) error {
	output, _ := utils.RunCommandOnHost("cat", "/etc/resolv.conf")
	err := utils.WriteToFile(hostDNSFile, output)
	if err != nil {
		return err
	}

	output, _ = utils.RunCommandOnContainer("cat", "/etc/resolv.conf")
	err = utils.WriteToFile(containerDNSFile, output)
	if err != nil {
		return err
	}

	return nil
}

func processDNS(files []string, DNSDiagnosticFile string) error {
	f, _ := os.OpenFile(DNSDiagnosticFile, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	var dnsDiagnosticData []dnsDiagnosticDatum

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

		dataPoint := dnsDiagnosticDatum{
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
