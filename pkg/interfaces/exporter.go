package interfaces

// Exporter defines interface for an exporter
type Exporter interface {
	Export(DataProducer) error
}
