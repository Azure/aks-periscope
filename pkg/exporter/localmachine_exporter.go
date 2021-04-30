package exporter

import (
	"archive/zip"
	"fmt"
	"log"
	"os"

	"github.com/Azure/aks-periscope/pkg/interfaces"
)

const (
	max_Container_Name_Length = 63
)

// LocalMachineExporter defines an Local Machine Exporter
type LocalMachineExporter struct{}

var _ interfaces.Exporter = &LocalMachineExporter{}

// Export implements the interface method
func (exporter *LocalMachineExporter) Export(files []string) error {
	outFile, err := os.Create("test.zip")
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	// Create a zip writer on top of the file writer
	zipWriter := zip.NewWriter(outFile)

	// Add files to archive
	// We use some hard coded data to demonstrate,
	// but you could iterate through all the files
	// in a directory and pass the name and contents
	// of each file, or you can take data from your
	// program and write it write in to the archive
	// without
	var filesToArchive = []struct {
		Name, Body string
	}{
		{"test.txt", "String contents of file"},
		{"test2.txt", "\x61\x62\x63\n"},
	}
	fmt.Printf("I am here")
	// Create and write files to the archive, which in turn
	// are getting written to the underlying writer to the
	// .zip file we created at the beginning
	for _, file := range filesToArchive {
		fileWriter, err := zipWriter.Create(file.Name)
		if err != nil {
			log.Fatal(err)
		}
		_, err = fileWriter.Write([]byte(file.Body))
		if err != nil {
			log.Fatal(err)
		}
	}

	// Clean up
	err = zipWriter.Close()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
