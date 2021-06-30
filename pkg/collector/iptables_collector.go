package collector

import (
	"github.com/Azure/aks-periscope/pkg/utils"
)

// IPTablesCollector defines a IPTables Collector struct
type IPTablesCollector struct {
	data map[string]string
}

// NewIPTablesCollector is a constructor
func NewIPTablesCollector() *IPTablesCollector {
	return &IPTablesCollector{
		data: make(map[string]string),
	}
}

func (collector *IPTablesCollector) GetName() string {
	return "iptables"
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

func (collector *IPTablesCollector) GetData() map[string]string {
	return collector.data
}
