package exporter

import (
	"archive/zip"
	"bytes"

	"github.com/Azure/aks-periscope/pkg/interfaces"
)

func Zip(data []interfaces.DataProducer) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	z := zip.NewWriter(buffer)
	defer z.Close()

	for _, prd := range data {
		for name, data := range prd.GetData() {
			dataf, err := z.Create(prd.GetName() + "/" + name)
			if err != nil {
				return nil, err
			}

			_, err = dataf.Write([]byte(data))
			if err != nil {
				return nil, err
			}
		}
	}

	z.Flush()

	return buffer, nil
}
