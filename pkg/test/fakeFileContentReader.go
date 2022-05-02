package test

import "fmt"

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
