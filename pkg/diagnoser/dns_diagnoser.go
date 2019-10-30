package diagnoser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

type dnsDiagnosticDatum struct {
	HostName    string   `json:"HostName"`
	LeveL       string   `json:"Level"`
	NameServers []string `json:"NameServer"`
	Custom      bool     `json:"Custom"`
}

// DNSDiagnoser defines a DNS Diagnoser struct
type DNSDiagnoser struct {
	BaseDiagnoser
	dnsCollector *collector.DNSCollector
}

var _ interfaces.Diagnoser = &DNSDiagnoser{}

// NewDNSDiagnoser is a constructor
func NewDNSDiagnoser(dnsCollector *collector.DNSCollector, exporter interfaces.Exporter) *DNSDiagnoser {
	return &DNSDiagnoser{
		BaseDiagnoser: BaseDiagnoser{
			diagnoserType: DNS,
			exporter:      exporter,
		},
		dnsCollector: dnsCollector,
	}
}

// Diagnose implements the interface method
func (diagnoser *DNSDiagnoser) Diagnose() error {
	hostName, err := utils.GetHostName()
	rootPath, err := utils.CreateDiagnosticDir()
	if err != nil {
		return err
	}

	dnsDiagnosticFile := filepath.Join(rootPath, diagnoser.GetName())

	f, err := os.OpenFile(dnsDiagnosticFile, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		return fmt.Errorf("Fail to open file %s: %+v", dnsDiagnosticFile, err)
	}

	dnsDiagnosticData := []dnsDiagnosticDatum{}
	for _, file := range diagnoser.dnsCollector.GetCollectorFiles() {
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
			HostName:    hostName,
			LeveL:       dnsLevel,
			NameServers: dns,
			Custom:      isCustom,
		}
		dnsDiagnosticData = append(dnsDiagnosticData, dataPoint)

	}

	dataBytes, err := json.Marshal(dnsDiagnosticData)
	if err != nil {
		return fmt.Errorf("Fail to marshal data: %+v", err)
	}

	_, err = f.WriteString(string(dataBytes))
	if err != nil {
		return fmt.Errorf("Fail to write data to file: %+v", err)
	}

	diagnoser.AddToDiagnoserFiles(dnsDiagnosticFile)

	err = utils.WriteToCRD(dnsDiagnosticFile, diagnoser.GetName())
	if err != nil {
		return fmt.Errorf("Fail to write file %s to CRD: %+v", dnsDiagnosticFile, err)
	}

	return nil
}
