package tests

import (
	"fmt"
	"testing"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/exporter"
)

func TestHelm(t *testing.T) {
	fmt.Printf("I am here")
	exporter := &exporter.AzureBlobExporter{}
	//collectors := []interfaces.Collector{}
	helmCollector := collector.NewHelmCollector(exporter)
	err := helmCollector.Collect()
	if err != nil {
		fmt.Printf("Error: %s", err)
	}

}
