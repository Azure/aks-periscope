package collector

import (
	"fmt"
	"runtime"

	"github.com/Azure/aks-periscope/pkg/utils"
)

// DNSCollector defines a DNS Collector struct
type DNSCollector struct {
	data map[string]string
}

// NewDNSCollector is a constructor
func NewDNSCollector() *DNSCollector {
	return &DNSCollector{
		data: make(map[string]string),
	}
}

func (collector *DNSCollector) GetName() string {
	return "dns"
}

func (collector *DNSCollector) CheckSupported() error {
	// NOTE: This *might* be achievable in Windows using APIs that query the registry, see:
	// https://kubernetes.io/docs/setup/production-environment/windows/intro-windows-in-kubernetes/#networking
	// But for now it's restricted to Linux containers only, in which we can read `resolv.conf`.
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
	}

	return nil
}

// Collect implements the interface method
func (collector *DNSCollector) Collect() error {
	output, err := utils.ReadFileContent("/etchostlogs/resolv.conf")
	if err != nil {
		output = err.Error()
	}

	collector.data["virtualmachine"] = output

	output, err = utils.ReadFileContent("/etc/resolv.conf")
	if err != nil {
		output = err.Error()
	}

	collector.data["kubernetes"] = output

	return nil
}

func (collector *DNSCollector) GetData() map[string]string {
	return collector.data
}
