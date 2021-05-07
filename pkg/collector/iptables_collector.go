package collector

import (
	"path/filepath"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// IPTablesCollector defines a IPTables Collector struct
type IPTablesCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &IPTablesCollector{}

// NewIPTablesCollector is a constructor
func NewIPTablesCollector(exporters []interfaces.Exporter) *IPTablesCollector {
	return &IPTablesCollector{
		BaseCollector: BaseCollector{
			collectorType: IPTables,
			exporters:      exporters,
		},
	}
}

// Collect implements the interface method
func (collector *IPTablesCollector) Collect() error {
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	iptablesFile := filepath.Join(rootPath, collector.GetName())

	output, err := utils.RunCommandOnHost("iptables", "-t", "nat", "-L")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(iptablesFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(iptablesFile)

	return nil
}
