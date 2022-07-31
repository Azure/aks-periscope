package interfaces

import "io"

type FileSystemAccessor interface {
	GetFileReader(filePath string) (io.ReadCloser, error)
	FileExists(filePath string) (bool, error)
	GetFileSize(filePath string) (int64, error)
	ListFiles(directoryPath string) ([]string, error)
}
