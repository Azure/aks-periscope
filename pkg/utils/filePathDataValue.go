package utils

import (
	"io"

	"github.com/Azure/aks-periscope/pkg/interfaces"
)

type FilePathDataValue struct {
	fileSystem interfaces.FileSystemAccessor
	filePath   string
	fileSize   int64
}

func NewFilePathDataValue(fileSystem interfaces.FileSystemAccessor, filePath string, fileSize int64) *FilePathDataValue {
	return &FilePathDataValue{
		fileSystem: fileSystem,
		filePath:   filePath,
		fileSize:   fileSize,
	}
}

func (v *FilePathDataValue) GetLength() int64 {
	return v.fileSize
}

func (v *FilePathDataValue) GetReader() (io.ReadCloser, error) {
	return v.fileSystem.GetFileReader(v.filePath)
}
