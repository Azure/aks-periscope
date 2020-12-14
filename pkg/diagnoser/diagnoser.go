package diagnoser

import (
	"github.com/Azure/aks-periscope/pkg/interfaces"
)

// Type defines Diagnoser Type
type Type int

const (
	// NetworkConfig defines NetworkConfig Diagnoser Type
	NetworkConfig Type = iota
	// NetworkOutbound defines NetworkOutbound Diagnoser Type
	NetworkOutbound
)

// Name returns type name
func (t Type) name() string {
	return [...]string{"networkconfig", "networkoutbound"}[t]
}

// BaseDiagnoser defines Base Diagnoser
type BaseDiagnoser struct {
	diagnoserType  Type
	diagnoserFiles []string
	exporter       interfaces.Exporter
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
	if b.exporter != nil {
		return b.exporter.Export(b.diagnoserFiles)
	}

	return nil
}
