package exporter

import (
	"archive/zip"
	"bytes"
	"io"
	"log"

	"github.com/Azure/aks-periscope/pkg/interfaces"
)

func Zip(data []interfaces.DataProducer) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	z := zip.NewWriter(buffer)
	defer z.Close()

	for _, prd := range data {
		for name, value := range prd.GetData() {
			key := prd.GetName() + "/" + name
			dataf, err := z.Create(key)
			if err != nil {
				// If there's an error creating one value, log the error and continue.
				// We don't this to prevent all the other logs from being exported.
				log.Printf("Error creating zip entry %q: %v", key, err)
				continue
			}

			err = func() error {
				valueReader, err := value.GetReader()
				if err != nil {
					return err
				}

				defer valueReader.Close()

				_, err = io.Copy(dataf, valueReader)
				return err
			}()

			if err != nil {
				// If there's an error writing one value, log the error and continue.
				// This will leave the entry in the zip empty but allow export of other entries.
				log.Printf("Error writing zip entry %q: %v", key, err)
				continue
			}
		}
	}

	z.Flush()

	return buffer, nil
}
