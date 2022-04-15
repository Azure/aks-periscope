package utils

import (
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
