package interfaces

import "io"

// Exporter defines interface for an exporter
type Exporter interface {
	GetName() string
	Export(DataProducer) error
	ExportReader(string, io.ReadSeeker) error
}
