package utils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type FileContentReader struct{}

// NewFileContentReader is a constructor
func NewFileContentReader() *FileContentReader {
	return &FileContentReader{}
}

func (reader *FileContentReader) GetFileContent(filePath string) (string, error) {
	output, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	defer output.Close()

	b, err := ioutil.ReadAll(output)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (reader *FileContentReader) FileExists(filePath string) (bool, error) {
	if _, err := os.Stat(filePath); err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, fmt.Errorf("error checking existence of file %s: %w", filePath, err)
	}
}

func (reader *FileContentReader) ListFiles(directoryPath string) ([]string, error) {
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
