package exporter

import (
	"testing"
)

var getStorageContainerNameTests = []struct {
	apiServerFqdn string
	containerName string
}{
	{"dakydd-test-eastus-dns-d0daedb9.hcp.eastus.azmk8s.io", "dakydd-test-eastus-dns-d0daedb9"},
}

// TestGetNonKINDStorageContainerName get storage container name for non kind cluster
func TestGetNonKINDStorageContainerName(t *testing.T) {
	for _, tt := range getStorageContainerNameTests {
		t.Run(tt.apiServerFqdn, func(t *testing.T) {
			var blobExporter = &AzureBlobExporter{}
			containerName, _ := blobExporter.GetNonKINDStorageContainerName(tt.apiServerFqdn)

			if containerName != tt.containerName {
				t.Errorf("Sprintf(%q, &blobExporter) => %q, want %q",
					tt.apiServerFqdn, containerName, tt.containerName)
			}
		})
	}
}
