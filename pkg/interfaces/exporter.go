package interfaces

import "io"

// Exporter defines interface for an exporter
type Exporter interface {
	Export(DataProducer) error
	ExportReader(string, io.ReadSeeker) error
}
