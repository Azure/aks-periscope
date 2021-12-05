package exporter

import (
	"archive/zip"
	"bytes"
	"sync"

	"github.com/Azure/aks-periscope/pkg/interfaces"
)

func Zip(wg *sync.WaitGroup, data []interfaces.DataProducer) (*bytes.Buffer, error) {
	defer wg.Done()
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
