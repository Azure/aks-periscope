package exporter

import (
	"archive/zip"
	"io"
	"io/ioutil"
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

	lsts, err := ioutil.ReadDir("./")

	if err != nil {

		log.Fatal(err)
	}

	for _, f := range lsts {

		log.Printf(f.Name())
	}
	output := "done.zip"

	if err = ZipFiles(output, files); err != nil {
		panic(err)
	}
	log.Printf("Zipped File: %s", output)
	return nil
}
func ZipFiles(filename string, files []string) error {

	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {
		if err = AddFileToZip(zipWriter, file); err != nil {
			return err
		}
	}
	return nil
}

func AddFileToZip(zipWriter *zip.Writer, filename string) error {

	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Using FileInfoHeader() above only uses the basename of the file. If we want
	// to preserve the folder structure we can overwrite this with the full path.
	header.Name = filename

	// Change to deflate to gain better compression
	// see http://golang.org/pkg/archive/zip/#pkg-constants
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}
