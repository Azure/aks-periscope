package collector

import (
	"fmt"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// DNSCollector defines a DNS Collector struct
type DNSCollector struct {
	data        map[string]string
	runtimeInfo *utils.RuntimeInfo
	filePaths   *utils.KnownFilePaths
	fileReader  interfaces.FileContentReader
}

// NewDNSCollector is a constructor
func NewDNSCollector(runtimeInfo *utils.RuntimeInfo, filePaths *utils.KnownFilePaths, fileReader interfaces.FileContentReader) *DNSCollector {
	return &DNSCollector{
		data:        make(map[string]string),
		runtimeInfo: runtimeInfo,
		filePaths:   filePaths,
		fileReader:  fileReader,
	}
}

func (collector *DNSCollector) GetName() string {
	return "dns"
}

func (collector *DNSCollector) CheckSupported() error {
	// NOTE: This *might* be achievable in Windows using APIs that query the registry, see:
	// https://kubernetes.io/docs/setup/production-environment/windows/intro-windows-in-kubernetes/#networking
	// But for now it's restricted to Linux containers only, in which we can read `resolv.conf`.
	if collector.runtimeInfo.OSIdentifier != "linux" {
		return fmt.Errorf("unsupported OS: %s", collector.runtimeInfo.OSIdentifier)
	}

	return nil
}

// Collect implements the interface method
func (collector *DNSCollector) Collect() error {
	output, err := collector.fileReader.GetFileContent(collector.filePaths.ResolvConfHost)
	if err != nil {
		output = err.Error()
	}

	collector.data["virtualmachine"] = output

	output, err = collector.fileReader.GetFileContent(collector.filePaths.ResolvConfContainer)
	if err != nil {
		output = err.Error()
	}

	collector.data["kubernetes"] = output

	return nil
}

func (collector *DNSCollector) GetData() map[string]string {
	return collector.data
}
