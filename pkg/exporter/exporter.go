package exporter

// Type defines Exporter Type
type Type int

const (
	// AzureBlob defines AzureBlob exporter Type
	AzureBlob Type = iota
)

// Name returns type name
func (t Type) name() string {
	return [...]string{"azureblob"}[t]
}

// BaseExporter defines Base Exporter
type BaseExporter struct {
	exporterType Type
}

// GetName gets exporter name
func (b *BaseExporter) GetName() string {
	return b.exporterType.name()
}
