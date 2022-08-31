package test

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// FakeFileSystem can be used to test code that uses the FileSystemAccessor interface to
// access the file system.
type FakeFileSystem struct {
	lookup     map[string]string
	errorFiles map[string]error
	lock       sync.RWMutex
}

// NewFakeFileSystem creates a FileSystemAccessor based on a map where the keys represent
// file paths and the values represent file content.
func NewFakeFileSystem(lookup map[string]string) *FakeFileSystem {
	return &FakeFileSystem{
		lookup:     lookup,
		errorFiles: map[string]error{},
		lock:       sync.RWMutex{},
	}
}

// GetFileReader implements the FileSystemAccessor interface
func (ffs *FakeFileSystem) GetFileReader(path string) (io.ReadCloser, error) {
	ffs.lock.RLock()
	defer ffs.lock.RUnlock()

	content, ok := ffs.lookup[path]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	if err := ffs.getError(path); err != nil {
		return nil, err
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

// FileExists implements the FileSystemAccessor interface
func (ffs *FakeFileSystem) FileExists(path string) (bool, error) {
	ffs.lock.RLock()
	defer ffs.lock.RUnlock()

	if err := ffs.getError(path); err != nil {
		return false, err
	}
	_, ok := ffs.lookup[path]
	return ok, nil
}

// GetFileSize implements the FileSystemAccessor interface
func (ffs *FakeFileSystem) GetFileSize(path string) (int64, error) {
	ffs.lock.RLock()
	defer ffs.lock.RUnlock()

	if err := ffs.getError(path); err != nil {
		return 0, err
	}
	content, ok := ffs.lookup[path]
	if !ok {
		return 0, fmt.Errorf("file not found: %s", path)
	}

	return int64(len(content)), nil
}

// ListFiles implements the FileSystemAccessor interface
func (ffs *FakeFileSystem) ListFiles(directoryPath string) ([]string, error) {
	ffs.lock.RLock()
	defer ffs.lock.RUnlock()

	files := []string{}
	if err := ffs.getError(directoryPath); err != nil {
		return files, err
	}
	for path := range ffs.lookup {
		if strings.HasPrefix(path, directoryPath+"/") {
			files = append(files, path)
		}
	}
	return files, nil
}

func (ffs *FakeFileSystem) SetFileAccessError(path string, err error) {
	ffs.lock.Lock()
	defer ffs.lock.Unlock()

	ffs.errorFiles[path] = err
}

func (ffs *FakeFileSystem) AddOrUpdateFile(path, content string) {
	ffs.lock.Lock()
	defer ffs.lock.Unlock()

	ffs.lookup[path] = content
}

func (ffs *FakeFileSystem) DeleteFile(path string) {
	ffs.lock.Lock()
	defer ffs.lock.Unlock()

	delete(ffs.lookup, path)
}

func (ffs *FakeFileSystem) getError(path string) error {
	if err, ok := ffs.errorFiles[path]; ok {
		return err
	}
	return nil
}
