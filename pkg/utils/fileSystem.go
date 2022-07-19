package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type FileSystem struct{}

func NewFileSystem() *FileSystem {
	return &FileSystem{}
}

func (fs *FileSystem) GetFileReader(filePath string) (io.ReadCloser, error) {
	return os.Open(filePath)
}

func (fs *FileSystem) FileExists(filePath string) (bool, error) {
	if _, err := os.Stat(filePath); err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, fmt.Errorf("error checking existence of file %s: %w", filePath, err)
	}
}

func (fs *FileSystem) GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("error getting file info for %s: %w", filePath, err)
	}

	return info.Size(), nil
}

func (fs *FileSystem) ListFiles(directoryPath string) ([]string, error) {
	paths := []string{}
	pathAdder := func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			// Always use forward-slash-separated paths for consistency
			paths = append(paths, filepath.ToSlash(path))
		}
		return err
	}

	if err := filepath.Walk(directoryPath, pathAdder); err != nil {
		return paths, fmt.Errorf("error listing files in %s: %w", directoryPath, err)
	}

	return paths, nil
}
