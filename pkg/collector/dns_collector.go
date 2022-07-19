package collector

import (
	"fmt"
	"io/ioutil"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// DNSCollector defines a DNS Collector struct
type DNSCollector struct {
	HostConf      string
	ContainerConf string
	runtimeInfo   *utils.RuntimeInfo
	filePaths     *utils.KnownFilePaths
	fileSystem    interfaces.FileSystemAccessor
}

// NewDNSCollector is a constructor
func NewDNSCollector(runtimeInfo *utils.RuntimeInfo, filePaths *utils.KnownFilePaths, fileSystem interfaces.FileSystemAccessor) *DNSCollector {
	return &DNSCollector{
		HostConf:      "",
		ContainerConf: "",
		runtimeInfo:   runtimeInfo,
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
	if collector.runtimeInfo.OSIdentifier != "linux" {
		return fmt.Errorf("unsupported OS: %s", collector.runtimeInfo.OSIdentifier)
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
	reader, err := collector.fileSystem.GetFileReader(filePath)
	if err != nil {
		return err.Error()
	}

	defer reader.Close()

	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return err.Error()
	}

	return string(content)
}

func (collector *DNSCollector) GetData() map[string]interfaces.DataValue {
	return map[string]interfaces.DataValue{
		"virtualmachine": utils.NewStringDataValue(collector.HostConf),
		"kubernetes":     utils.NewStringDataValue(collector.ContainerConf),
	}
}
