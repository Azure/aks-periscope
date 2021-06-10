package exporter

import (
	"archive/zip"
	"io"
	"log"
	"os"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// LocalMachineExporter defines an Local Machine Exporter
type LocalMachineExporter struct{}

var _ interfaces.Exporter = &LocalMachineExporter{}

// Export implements the interface method
func (exporter *LocalMachineExporter) Export(files []string) error {
	hostName, err := utils.GetHostName()
	if err != nil {
		return err
	}

	outFile, err := os.Create(hostName + ".zip")
	if err != nil {
		return err
	}

	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	for _, file := range files {
		log.Printf("File %s is being exported to local machine", file)
		fileToZip, err := os.Open(file)
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
		header.Name = file

		// Change to deflate to gain better compression
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, fileToZip)
		if err != nil {
			return err
		}
	}
	return nil
}
