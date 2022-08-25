package utils

import (
	"io"
	"os"
	"path"
	"testing"

	"github.com/google/uuid"
)

func setup(t *testing.T) (*os.File, func()) {
	file, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	teardown := func() {
		file.Close()
		os.Remove(file.Name())
	}

	return file, teardown
}

func TestGetFileReaderForExistingFile(t *testing.T) {
	testFile, teardown := setup(t)
	defer teardown()

	const expectedContent = "Test File Content"

	_, err := testFile.Write([]byte(expectedContent))
	if err != nil {
		t.Errorf("failed to write to file %s: %s", testFile.Name(), expectedContent)
	}

	fs := NewFileSystem()
	actualContent, err := GetContent(func() (io.ReadCloser, error) { return fs.GetFileReader(testFile.Name()) })
	if err != nil {
		t.Errorf("error reading content from %s", testFile.Name())
	}

	if actualContent != expectedContent {
		t.Errorf("unexpected file content.\nExpected '%s'\nFound '%s'", expectedContent, actualContent)
	}
}

func TestGetFileContentForMissingFile(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Errorf("error getting current directory: %v", err)
	}

	missingFilePath := path.Join(cwd, uuid.New().String())

	fs := NewFileSystem()
	_, err = fs.GetFileReader(missingFilePath)
	if err == nil {
		t.Errorf("no error reading missing file %s", missingFilePath)
	}
}

func TestFileExistsForExistingFile(t *testing.T) {
	testFile, teardown := setup(t)
	defer teardown()

	fs := NewFileSystem()
	exists, err := fs.FileExists(testFile.Name())

	if err != nil {
		t.Errorf("error checking existence of file %s", testFile.Name())
	}

	if !exists {
		t.Errorf("file exists but FileExists returned false")
	}
}

func TestFileExistsForMissingFile(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Errorf("error getting current directory: %v", err)
	}

	missingFilePath := path.Join(cwd, uuid.New().String())

	fs := NewFileSystem()
	exists, err := fs.FileExists(missingFilePath)

	if err != nil {
		t.Errorf("error checking existence of missing file %s", missingFilePath)
	}

	if exists {
		t.Errorf("file does not exist but FileExists returned true")
	}
}
