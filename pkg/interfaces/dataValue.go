package interfaces

import "io"

type DataValue interface {
	GetLength() int64

	GetReader() (io.ReadCloser, error)
}
