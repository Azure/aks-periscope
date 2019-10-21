package action

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
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

	rootPath, err := utils.CreateCollectorDir(action.GetName())
	if err != nil {
		return err
	}

	hostDNSFile := filepath.Join(rootPath, "host")
	containerDNSFile := filepath.Join(rootPath, "container")

	output, err := utils.RunCommandOnHost("cat", "/etc/resolv.conf")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(hostDNSFile, output)
	if err != nil {
		return err
	}

	action.collectFiles = append(action.collectFiles, hostDNSFile)

	output, err = utils.RunCommandOnContainer("cat", "/etc/resolv.conf")
	if err != nil {
		return err
	}

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

	rootPath, err := utils.CreateDiagnosticDir()
	if err != nil {
		return err
	}

	dnsDiagnosticFile := filepath.Join(rootPath, action.GetName())

	f, err := os.OpenFile(dnsDiagnosticFile, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		return fmt.Errorf("Fail to open file %s: %+v", dnsDiagnosticFile, err)
	}

	dnsDiagnosticData := []dnsDiagnosticDatum{}
	for _, file := range action.collectFiles {
		t, err := os.Open(file)
		defer t.Close()
		if err != nil {
			return fmt.Errorf("Fail to open file %s: %+v", file, err)
		}

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
		dataBytes, err := json.Marshal(dataPoint)
		if err != nil {
			return fmt.Errorf("Fail to marshal data: %+v", err)
		}

		_, err = f.WriteString(string(dataBytes) + "\n")
		if err != nil {
			return fmt.Errorf("Fail to write data to file: %+v", err)
		}
	}

	action.processFiles = append(action.processFiles, dnsDiagnosticFile)

	err = utils.WriteToCRD(dnsDiagnosticFile, action.GetName())
	if err != nil {
		return fmt.Errorf("Fail to write file %s to CRD: %+v", dnsDiagnosticFile, err)
	}

	return nil
}

// Export implements the interface method
func (action *dnsAction) Export() error {
	if action.exporter != nil {
		return action.exporter.Export(append(action.collectFiles, action.processFiles...))
	}

	return nil
}
