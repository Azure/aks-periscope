package test

import (
	"fmt"
	"strings"
)

// FakeFileContentReader can be used to test code that uses the FileContentReader interface to
// access the file system.
type FakeFileContentReader struct {
	lookup map[string]string
}

// NewFakeFileContentReader creates a FileContentReader based on a map where the keys represent
// file paths and the values represent file content.
func NewFakeFileContentReader(lookup map[string]string) *FakeFileContentReader {
	return &FakeFileContentReader{
		lookup: lookup,
	}
}

// GetFileContent implements the FileContentReader interface
func (reader *FakeFileContentReader) GetFileContent(path string) (string, error) {
	content, ok := reader.lookup[path]
	if !ok {
		return "", fmt.Errorf("file not found: %s", path)
	}
	return content, nil
}

// FileExists implements the FileContentReader interface
func (reader *FakeFileContentReader) FileExists(path string) (bool, error) {
	_, ok := reader.lookup[path]
	return ok, nil
}

// ListFiles implements the FileContentReader interface
func (reader *FakeFileContentReader) ListFiles(directoryPath string) ([]string, error) {
	files := []string{}
	for path := range reader.lookup {
		if strings.HasPrefix(path, directoryPath+"/") {
			files = append(files, path)
		}
	}
	return files, nil
}

func (reader *FakeFileContentReader) AddOrUpdateFile(path, content string) {
	reader.lookup[path] = content
}

func (reader *FakeFileContentReader) DeleteFile(path string) {
	delete(reader.lookup, path)
}
