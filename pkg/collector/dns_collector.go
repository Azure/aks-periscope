package collector

import (
	"path/filepath"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// DNSCollector defines a DNS Collector struct
type DNSCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &DNSCollector{}

// NewDNSCollector is a constructor
func NewDNSCollector(exporter interfaces.Exporter) *DNSCollector {
	return &DNSCollector{
		BaseCollector: BaseCollector{
			collectorType: DNS,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *DNSCollector) Collect() error {
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	hostDNSFile := filepath.Join(rootPath, "host")
	containerDNSFile := filepath.Join(rootPath, "container")

	output, err := utils.RunCommandOnHost("cat", "/etc/resolv.conf")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(hostDNSFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(hostDNSFile)

	output, err = utils.RunCommandOnContainer("cat", "/etc/resolv.conf")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(containerDNSFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(containerDNSFile)

	return nil
}
