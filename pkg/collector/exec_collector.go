package collector

import (
	"github.com/Azure/aks-periscope/pkg/interfaces"
)

// ExecCollector defines a Exec Collector struct
type ExecCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &ExecCollector{}

// ExecCollector is a constructor
func NewExecCollector(exporter interfaces.Exporter) *ExecCollector {
	return &ExecCollector{
		BaseCollector: BaseCollector{
			collectorType: Exec,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *ExecCollector) Collect() error {
	return nil
}
