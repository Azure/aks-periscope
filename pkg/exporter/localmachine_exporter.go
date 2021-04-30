package exporter

import (
	"log"

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
		log.Printf("Filename: %s", file)
		//err := os.Rename(file, "C:/Users/sophiezhao/.azure/cliextensions/connectedk8s"+file)
		//os.makedirs(os.path.dirname(target), exist_ok=True)
		//if err != nil {
		//return fmt.Errorf("Error: %s", err)
		//}
	}
	return nil
}
