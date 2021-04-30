package exporter

import (
	"fmt"
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
	for _, file := range files {

		err := os.Rename(file, "C:/Users/sophiezhao/.azure/cliextensions/connectedk8s"+file)
		if err != nil {
			return fmt.Errorf("fail to write file to directory")
		}
	}
	return nil
}
