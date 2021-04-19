package collector

import (
	"github.com/Azure/aks-periscope/pkg/interfaces"
)

// CustomResourceCollector defines a CustomResources Collector struct
type CustomResourceCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &CustomResourceCollector{}

// CustomResourceCollector is a constructor
func NewCustomResourceCollector(exporter interfaces.Exporter) *CustomResourceCollector {
	return &CustomResourceCollector{
		BaseCollector: BaseCollector{
			collectorType: CustomResource,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *CustomResourceCollector) Collect() error {
	return nil
}
