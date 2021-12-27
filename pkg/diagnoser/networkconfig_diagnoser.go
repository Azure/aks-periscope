package diagnoser

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/aks-periscope/pkg/collector"
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
	dnsCollector        *collector.DNSCollector
	kubeletCmdCollector *collector.KubeletCmdCollector
	data                map[string]string
}

// NewNetworkConfigDiagnoser is a constructor
func NewNetworkConfigDiagnoser(dnsCollector *collector.DNSCollector, kubeletCmdCollector *collector.KubeletCmdCollector) *NetworkConfigDiagnoser {
	return &NetworkConfigDiagnoser{
		dnsCollector:        dnsCollector,
		kubeletCmdCollector: kubeletCmdCollector,
		data:                make(map[string]string),
	}
}

func (collector *NetworkConfigDiagnoser) GetName() string {
	return "networkconfig"
}

// Diagnose implements the interface method
func (diagnoser *NetworkConfigDiagnoser) Diagnose() error {
	hostName, err := utils.GetHostName()
	if err != nil {
		return err
	}

	networkConfigDiagnosticData := networkConfigDiagnosticDatum{HostName: hostName}
	for key, data := range diagnoser.dnsCollector.GetData() {
		var dns []string
		words := strings.Split(data, " ")
		for i := range words {
			if words[i] == "nameserver" {
				dns = append(dns, strings.TrimSuffix(words[i+1], "\n"))
			}
		}

		if key == "virtualmachine" {
			networkConfigDiagnosticData.VirtualMachineDNS = dns
		}

		if key == "kubernetes" {
			networkConfigDiagnosticData.KubernetesDNS = dns
		}
	}

	for _, data := range diagnoser.kubeletCmdCollector.GetData() {
		parts := strings.Split(data, " ")
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
		return fmt.Errorf("marshal data from NetworkConfig Diagnoser: %w", err)
	}

	diagnoser.data["networkconfig"] = string(dataBytes)

	return nil
}

func (collector *NetworkConfigDiagnoser) GetData() map[string]string {
	return collector.data
}
