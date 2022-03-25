package collector

import (
	"fmt"
	"runtime"

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

func (collector *IPTablesCollector) CheckSupported() error {
	// There's no obvious alternative to `iptables` on Windows.
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
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

func (collector *IPTablesCollector) GetData() map[string]string {
	return collector.data
}
