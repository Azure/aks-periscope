package collector

import (
	"fmt"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// DNSCollector defines a DNS Collector struct
type DNSCollector struct {
	HostConf      string
	ContainerConf string
	osIdentifier  utils.OSIdentifier
	filePaths     *utils.KnownFilePaths
	fileSystem    interfaces.FileSystemAccessor
}

// NewDNSCollector is a constructor
func NewDNSCollector(osIdentifier utils.OSIdentifier, filePaths *utils.KnownFilePaths, fileSystem interfaces.FileSystemAccessor) *DNSCollector {
	return &DNSCollector{
		HostConf:      "",
		ContainerConf: "",
		osIdentifier:  osIdentifier,
		filePaths:     filePaths,
		fileSystem:    fileSystem,
	}
}

func (collector *DNSCollector) GetName() string {
	return "dns"
}

func (collector *DNSCollector) CheckSupported() error {
	// NOTE: This *might* be achievable in Windows using APIs that query the registry, see:
	// https://kubernetes.io/docs/setup/production-environment/windows/intro-windows-in-kubernetes/#networking
	// But for now it's restricted to Linux containers only, in which we can read `resolv.conf`.
	if collector.osIdentifier != utils.Linux {
		return fmt.Errorf("unsupported OS: %s", collector.osIdentifier)
	}

	return nil
}

// Collect implements the interface method
func (collector *DNSCollector) Collect() error {
	collector.HostConf = collector.getConfFileContent(collector.filePaths.ResolvConfHost)
	collector.ContainerConf = collector.getConfFileContent(collector.filePaths.ResolvConfContainer)

	return nil
}

func (collector *DNSCollector) getConfFileContent(filePath string) string {
	content, err := utils.GetFileContent(collector.fileSystem, filePath)
	if err != nil {
		return err.Error()
	}

	return content
}

func (collector *DNSCollector) GetData() map[string]interfaces.DataValue {
	return map[string]interfaces.DataValue{
		"virtualmachine": utils.NewStringDataValue(collector.HostConf),
		"kubernetes":     utils.NewStringDataValue(collector.ContainerConf),
	}
}
