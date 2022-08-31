package collector

import (
	"fmt"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// IPTablesCollector defines a IPTables Collector struct
type IPTablesCollector struct {
	data         map[string]string
	osIdentifier utils.OSIdentifier
	runtimeInfo  *utils.RuntimeInfo
}

// NewIPTablesCollector is a constructor
func NewIPTablesCollector(osIdentifier utils.OSIdentifier, runtimeInfo *utils.RuntimeInfo) *IPTablesCollector {
	return &IPTablesCollector{
		data:         make(map[string]string),
		osIdentifier: osIdentifier,
		runtimeInfo:  runtimeInfo,
	}
}

func (collector *IPTablesCollector) GetName() string {
	return "iptables"
}

func (collector *IPTablesCollector) CheckSupported() error {
	// There's no obvious alternative to `iptables` on Windows.
	if collector.osIdentifier != utils.Linux {
		return fmt.Errorf("unsupported OS: %s", collector.osIdentifier)
	}

	if utils.Contains(collector.runtimeInfo.CollectorList, "connectedCluster") {
		return fmt.Errorf("not included because 'connectedCluster' is in COLLECTOR_LIST variable. Included values: %s", strings.Join(collector.runtimeInfo.CollectorList, " "))
	}

	return nil
}

// Collect implements the interface method
func (collector *IPTablesCollector) Collect() error {
	output, err := utils.RunCommandOnHost("iptables", "-t", "nat", "-L")
	if err != nil {
		return err
	}

	collector.data["iptables"] = output

	return nil
}

func (collector *IPTablesCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}
