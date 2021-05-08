package diagnoser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

type networkConfigDiagnosticDatum struct {
	HostName          string   `json:"HostName"`
	NetworkPlugin     string   `json:"NetworkPlugin"`
	VirtualMachineDNS []string `json:"VirtualMachineDNS"`
	KubernetesDNS     []string `json:"KubernetesDNS"`
	MaxPodsPerNode    int      `json:"MaxPodsPerNode"`
}

// NetworkConfigDiagnoser defines a NetworkConfig Diagnoser struct
type NetworkConfigDiagnoser struct {
	BaseDiagnoser
	dnsCollector        *collector.DNSCollector
	kubeletCmdCollector *collector.KubeletCmdCollector
}

var _ interfaces.Diagnoser = &NetworkConfigDiagnoser{}

// NewNetworkConfigDiagnoser is a constructor
func NewNetworkConfigDiagnoser(dnsCollector *collector.DNSCollector, kubeletCmdCollector *collector.KubeletCmdCollector, exporters []interfaces.Exporter) *NetworkConfigDiagnoser {
	return &NetworkConfigDiagnoser{
		BaseDiagnoser: BaseDiagnoser{
			diagnoserType: NetworkConfig,
			exporters:     exporters,
		},
		dnsCollector:        dnsCollector,
		kubeletCmdCollector: kubeletCmdCollector,
	}
}

// Diagnose implements the interface method
func (diagnoser *NetworkConfigDiagnoser) Diagnose() error {
	hostName, err := utils.GetHostName()
	rootPath, err := utils.CreateDiagnosticDir()
	if err != nil {
		return err
	}

	networkDiagnosticFile := filepath.Join(rootPath, diagnoser.GetName())

	f, err := os.OpenFile(networkDiagnosticFile, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		return fmt.Errorf("Fail to open file %s: %+v", networkDiagnosticFile, err)
	}

	networkConfigDiagnosticData := networkConfigDiagnosticDatum{HostName: hostName}
	for _, file := range diagnoser.dnsCollector.GetCollectorFiles() {
		t, err := os.Open(file)
		defer t.Close()
		if err != nil {
			return fmt.Errorf("Fail to open file %s: %+v", file, err)
		}

		dnsLevel := filepath.Base(file)
		var dns []string
		scanner := bufio.NewScanner(t)
		for scanner.Scan() {
			s := strings.Split(scanner.Text(), " ")
			if s[0] == "nameserver" {
				dns = append(dns, strings.TrimSuffix(s[1], "\n"))
			}
		}

		if dnsLevel == "virtualmachine" {
			networkConfigDiagnosticData.VirtualMachineDNS = dns
		}

		if dnsLevel == "kubernetes" {
			networkConfigDiagnosticData.KubernetesDNS = dns
		}
	}

	for _, file := range diagnoser.kubeletCmdCollector.GetCollectorFiles() {
		f, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("Fail to read file %s: %+v", file, err)
		}

		parts := strings.Split(string(f), " ")

		for _, part := range parts {
			if strings.HasPrefix(part, "--network-plugin=") {
				networkPlugin := part[17:]
				if networkPlugin == "cni" {
					networkPlugin = "azurecni"
				}

				networkConfigDiagnosticData.NetworkPlugin = networkPlugin
			}

			if strings.HasPrefix(part, "--max-pods=") {
				maxPodsPerNodeString := part[11:]
				maxPodsPerNode, _ := strconv.Atoi(maxPodsPerNodeString)
				networkConfigDiagnosticData.MaxPodsPerNode = maxPodsPerNode
			}
		}
	}

	dataBytes, err := json.Marshal(networkConfigDiagnosticData)
	if err != nil {
		return fmt.Errorf("Fail to marshal data: %+v", err)
	}

	_, err = f.WriteString(string(dataBytes))
	if err != nil {
		return fmt.Errorf("Fail to write data to file: %+v", err)
	}

	diagnoser.AddToDiagnoserFiles(networkDiagnosticFile)

	err = utils.WriteToCRD(networkDiagnosticFile, diagnoser.GetName())
	if err != nil {
		return fmt.Errorf("Fail to write file %s to CRD: %+v", networkDiagnosticFile, err)
	}

	return nil
}
