package exporter

import (
	"archive/zip"
	"bytes"
	"io"

	"github.com/Azure/aks-periscope/pkg/interfaces"
)

func Zip(data []interfaces.DataProducer) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	z := zip.NewWriter(buffer)
	defer z.Close()

	for _, prd := range data {
		for name, value := range prd.GetData() {
			dataf, err := z.Create(prd.GetName() + "/" + name)
			if err != nil {
				return nil, err
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
				return nil, err
			}
		}
	}

	z.Flush()

	return buffer, nil
}
