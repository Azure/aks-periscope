package exporter

import (
	"testing"
)

var getStorageContainerNameTests = []struct {
	apiServerFqdn string
	containerName string
}{
	{"standard-aks-fqdn-dns-d0daedb9.hcp.eastus.azmk8s.io", "standard-aks-fqdn-dns-d0daedb9"},
	{"aks-engine-fqdn.westeurope.cloudapp.azure.com", "aks-engine-fqdn-westeurope-cloudapp-azure-com"},
	{"additional.aks-engine-fqdn.db839748.eastus.cloudapp.azure.com", "additional-aks-engine-fqdn-db839748-eastus-cloudapp-azure-com"},
	{"10.255.255.5", "10-255-255-5"}, // aks-engine clusters will currently return an IPv4 address as what Periscope is calling the APIServerFQDN
	{"extra.super.duper.long.apiserverfqdn.that.has.more.than.63.characters", "extra-super-duper-long-apiserverfqdn-that-has-more-than-63-char"},
}

// TestGetAKSStorageContainerName get storage container name for non kind cluster
func TestGetAKSStorageContainerName(t *testing.T) {
	for _, tt := range getStorageContainerNameTests {
		t.Run(tt.apiServerFqdn, func(t *testing.T) {
			containerName, _ := getAKSStorageContainerName(tt.apiServerFqdn)

			if containerName != tt.containerName {
				t.Errorf("GetAKSStorageContainerName(%q) => %q, want %q",
					tt.apiServerFqdn, containerName, tt.containerName)
			}
		})
	}
}
