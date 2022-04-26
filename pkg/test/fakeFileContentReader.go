package test

import "fmt"

type FakeFileContentReader struct {
	lookup map[string]string
}

func (reader *FakeFileContentReader) GetFileContent(path string) (string, error) {
	content, ok := reader.lookup[path]
	if !ok {
		return "", fmt.Errorf("file not found: %s", path)
	}
	return content, nil
}

func NewFakeFileContentReader(lookup map[string]string) *FakeFileContentReader {
	return &FakeFileContentReader{
		lookup: lookup,
	}
}
