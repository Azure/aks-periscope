package diagnoser

import (
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/hashicorp/go-multierror"
)

// Type defines Diagnoser Type
type Type int

const (
	// NetworkConfig defines NetworkConfig Diagnoser Type
	NetworkConfig Type = iota
	// NetworkOutbound defines NetworkOutbound Diagnoser Type
	NetworkOutbound Type = iota
)

// Name returns type name
func (t Type) name() string {
	return [...]string{"networkconfig", "networkoutbound"}[t]
}

// BaseDiagnoser defines Base Diagnoser
type BaseDiagnoser struct {
	diagnoserType  Type
	diagnoserFiles []string
	exporters      []interfaces.Exporter
}

// GetName gets diagnoser name
func (b *BaseDiagnoser) GetName() string {
	return b.diagnoserType.name()
}

// AddToDiagnoserFiles adds a file to diagnoser files
func (b *BaseDiagnoser) AddToDiagnoserFiles(file string) {
	b.diagnoserFiles = append(b.diagnoserFiles, file)
}

// Export implements the interface method
func (b *BaseDiagnoser) Export() error {
	var result error
	for _, exporter := range b.exporters {
		if exporter != nil {
			if err := exporter.Export(b.diagnoserFiles); err != nil {
				result = multierror.Append(result, err)
			}
		}
	}
	return result
}
