package utils

import (
	"testing"
)

var parseAPIServerFQDNFromKubeConfigTests = []struct {
	kubeConfig    string
	APIServerFQDN string
}{
	{`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: dummyData==
    server: https://kind-control-plane:6443
  name: kind
contexts:
- context:
    cluster: kind
    user: system:node:kind-control-plane
  name: system:node:kind-control-plane@kind
current-context: system:node:kind-control-plane@kind
kind: Config
preferences: {}
users:
- name: system:node:kind-control-plane
  user:
    client-certificate: /var/lib/kubelet/pki/kubelet-client-current.pem
    client-key: /var/lib/kubelet/pki/kubelet-client-current.pem`,
		"kind-control-plane"},

	{`apiVersion: v1
kind: Config
clusters:
- name: localcluster
  cluster:
    certificate-authority: /etc/kubernetes/certs/ca.crt
    server: https://dakydd-test-eastus-dns-d0daedb9.hcp.eastus.azmk8s.io:443
users:
- name: client
  user:
    client-certificate: /etc/kubernetes/certs/client.crt
    client-key: /etc/kubernetes/certs/client.key
contexts:
- context:
    cluster: localcluster
    user: client
  name: localclustercontext
current-context: localclustercontext`,
		"dakydd-test-eastus-dns-d0daedb9.hcp.eastus.azmk8s.io"},
}

// GetStorageContainerName get storage container name
func TestParseAPIServerFQDNFromKubeConfig(t *testing.T) {
	for _, tt := range parseAPIServerFQDNFromKubeConfigTests {
		t.Run(tt.APIServerFQDN, func(t *testing.T) {
			APIServerFQDN, err := ParseAPIServerFQDNFromKubeConfig(tt.kubeConfig)
			if err == nil {
				t.Errorf("utils.TestParseAPIServerFQDNFromKubeConfig(%q) Error: %q, expected %q",
					tt.kubeConfig, err, tt.APIServerFQDN)
			}

			if APIServerFQDN != tt.APIServerFQDN {
				t.Errorf("utils.TestParseAPIServerFQDNFromKubeConfig(%q) => %q, want %q",
					tt.kubeConfig, APIServerFQDN, tt.APIServerFQDN)
			}
		})
	}
}
