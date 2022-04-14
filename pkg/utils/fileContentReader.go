package utils

import (
	"fmt"
	"io/ioutil"
	"os"
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

type FakeFileContentReader struct {
	lookup map[string]string
}

func (reader *FakeFileContentReader) GetFileContent(path string) (string, error) {
	content, ok := reader.lookup[path]
	if !ok {
		return "", fmt.Errorf("File not found: %s", path)
	}
	return content, nil
}

func NewFakeFileContentReader(lookup map[string]string) *FakeFileContentReader {
	return &FakeFileContentReader{
		lookup: lookup,
	}
}
