package test

import (
	"fmt"
	"io"
	"strings"
)

// FakeFileSystem can be used to test code that uses the FileSystemAccessor interface to
// access the file system.
type FakeFileSystem struct {
	lookup map[string]string
}

// NewFakeFileSystem creates a FileSystemAccessor based on a map where the keys represent
// file paths and the values represent file content.
func NewFakeFileSystem(lookup map[string]string) *FakeFileSystem {
	return &FakeFileSystem{
		lookup: lookup,
	}
}

// GetFileReader implements the FileSystemAccessor interface
func (ffs *FakeFileSystem) GetFileReader(path string) (io.ReadCloser, error) {
	content, ok := ffs.lookup[path]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

// FileExists implements the FileSystemAccessor interface
func (ffs *FakeFileSystem) FileExists(path string) (bool, error) {
	_, ok := ffs.lookup[path]
	return ok, nil
}

// GetFileSize implements the FileSystemAccessor interface
func (ffs *FakeFileSystem) GetFileSize(path string) (int64, error) {
	content, ok := ffs.lookup[path]
	if !ok {
		return 0, fmt.Errorf("file not found: %s", path)
	}

	return int64(len(content)), nil
}

// ListFiles implements the FileSystemAccessor interface
func (ffs *FakeFileSystem) ListFiles(directoryPath string) ([]string, error) {
	files := []string{}
	for path := range ffs.lookup {
		if strings.HasPrefix(path, directoryPath+"/") {
			files = append(files, path)
		}
	}
	return files, nil
}

func (ffs *FakeFileSystem) AddOrUpdateFile(path, content string) {
	ffs.lookup[path] = content
}

func (ffs *FakeFileSystem) DeleteFile(path string) {
	delete(ffs.lookup, path)
}
