package diagnoser

import (
	"encoding/json"
	"fmt"
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
	runtimeInfo         *utils.RuntimeInfo
	dnsCollector        *collector.DNSCollector
	kubeletCmdCollector *collector.KubeletCmdCollector
	data                map[string]string
}

// NewNetworkConfigDiagnoser is a constructor
func NewNetworkConfigDiagnoser(runtimeInfo *utils.RuntimeInfo, dnsCollector *collector.DNSCollector, kubeletCmdCollector *collector.KubeletCmdCollector) *NetworkConfigDiagnoser {
	return &NetworkConfigDiagnoser{
		runtimeInfo:         runtimeInfo,
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
	networkConfigDiagnosticData := networkConfigDiagnosticDatum{HostName: diagnoser.runtimeInfo.HostNodeName}

	networkConfigDiagnosticData.VirtualMachineDNS = diagnoser.getDns(diagnoser.dnsCollector.HostConf)
	networkConfigDiagnosticData.KubernetesDNS = diagnoser.getDns(diagnoser.dnsCollector.ContainerConf)

	parts := strings.Split(diagnoser.kubeletCmdCollector.KubeletCommand, " ")
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

	dataBytes, err := json.Marshal(networkConfigDiagnosticData)
	if err != nil {
		return fmt.Errorf("marshal data from NetworkConfig Diagnoser: %w", err)
	}

	diagnoser.data["networkconfig"] = string(dataBytes)

	return nil
}

func (diagnoser *NetworkConfigDiagnoser) getDns(confFileContent string) []string {
	var dns []string
	words := strings.Split(confFileContent, " ")
	for i := range words {
		if words[i] == "nameserver" {
			dns = append(dns, strings.TrimSuffix(words[i+1], "\n"))
		}
	}

	return dns
}

func (collector *NetworkConfigDiagnoser) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}
