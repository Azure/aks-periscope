package main

import (
	"fmt"
	"testing"

	"github.com/Azure/aks-periscope/pkg/exporter"
)

func TestZipAndExport(t *testing.T) {
	exporter := &exporter.AzureBlobExporter{}
	err := zipAndExport(exporter)

	fmt.Printf("some test here with: %v\n", err)
	t.Log("test test")

	result := err
	expected := -2
	if result == nil {
		t.Errorf("sub() test returned an unexpected result: got %v want %v", result, expected)
	}
}
