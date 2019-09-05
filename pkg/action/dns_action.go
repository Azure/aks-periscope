package action

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

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
	collectCountForProcess   int
	collectCountForExport    int
	exporter                 interfaces.Exporter
	collectFiles             []string
	processFiles             []string
}

var _ interfaces.Action = &dnsAction{}

// NewDNSAction is a constructor
func NewDNSAction(collectIntervalInSeconds int, collectCountForProcess int, collectCountForExport int, exporter interfaces.Exporter) interfaces.Action {
	return &dnsAction{
		name:                     "dns",
		collectIntervalInSeconds: collectIntervalInSeconds,
		collectCountForProcess:   collectCountForProcess,
		collectCountForExport:    collectCountForExport,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *dnsAction) GetName() string {
	return action.name
}

// GetName implements the interface method
func (action *dnsAction) GetCollectIntervalInSeconds() int {
	return action.collectIntervalInSeconds
}

// GetName implements the interface method
func (action *dnsAction) GetCollectCountForProcess() int {
	return action.collectCountForProcess
}

// GetName implements the interface method
func (action *dnsAction) GetCollectCountForExport() int {
	return action.collectCountForExport
}

// Collect implements the interface method
func (action *dnsAction) Collect() error {
	action.collectFiles = []string{}

	rootPath, _ := utils.CreateCollectorDir(action.GetName())
	hostDNSFile := filepath.Join(rootPath, "host")
	containerDNSFile := filepath.Join(rootPath, "container")

	output, _ := utils.RunCommandOnHost("cat", "/etc/resolv.conf")
	err := utils.WriteToFile(hostDNSFile, output)
	if err != nil {
		return err
	}

	action.collectFiles = append(action.collectFiles, hostDNSFile)

	output, _ = utils.RunCommandOnContainer("cat", "/etc/resolv.conf")
	err = utils.WriteToFile(containerDNSFile, output)
	if err != nil {
		return err
	}

	action.collectFiles = append(action.collectFiles, containerDNSFile)

	return nil
}

// Process implements the interface method
func (action *dnsAction) Process() error {
	action.processFiles = []string{}

	rootPath, _ := utils.CreateDiagnosticDir()
	dnsDiagnosticFile := filepath.Join(rootPath, action.GetName())

	f, _ := os.OpenFile(dnsDiagnosticFile, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	dnsDiagnosticData := []dnsDiagnosticDatum{}
	for _, file := range action.collectFiles {
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

	action.processFiles = append(action.processFiles, dnsDiagnosticFile)

	return nil
}

// Export implements the interface method
func (action *dnsAction) Export() error {
	if action.exporter != nil {
		return action.exporter.Export(append(action.collectFiles, action.processFiles...))
	}

	return nil
}
