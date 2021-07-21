package main

import (
	"errors"
	"log"
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
	. "github.com/onsi/gomega"
)

func TestGetHostNameSuccessCase(t *testing.T) {
	g := NewGomegaWithT(t)

	// Save current function and restore at the end:
	old := utils.GetHostNameFunc
	defer func() { utils.GetHostNameFunc = old }()

	utils.GetHostNameFunc = &utils.HostName{
		HostName: "aks-agentpool-20752274-vmss000000",
		Err:      nil,
	}

	// setup expectations
	// call the code we are testing
	hostname, _ := utils.GetHostName()

	// assert that the expectations were met
	g.Expect(hostname).To(BeElementOf("aks-agentpool-20752274-vmss000000"))
}

func TestGetHostNameFailureCase(t *testing.T) {
	g := NewGomegaWithT(t)

	// Save current function and restore at the end:
	old := utils.GetHostNameFunc
	defer func() { utils.GetHostNameFunc = old }()

	utils.GetHostNameFunc = &utils.HostName{
		HostName: "",
		Err:      errors.New("an error"),
	}

	// setup expectations
	// call the code we are testing
	_, err := utils.GetHostName()
	log.Printf("Error is %s", err)
	// assert that the expectations were met
	g.Expect(err).Should(HaveOccurred())
	g.Expect(err).To(BeElementOf(errors.New("Fail to get host name: an error")))

}

//table of tests for utils.ParseAPIServerFQDNFromKubeConfig
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

func TestParseAPIServerFQDNFromKubeConfig(t *testing.T) {
	g := NewWithT(t)
	for _, tt := range parseAPIServerFQDNFromKubeConfigTests {
		t.Run(tt.APIServerFQDN, func(t *testing.T) {
			//call the test function
			APIServerFQDN, err := utils.ParseAPIServerFQDNFromKubeConfig(tt.kubeConfig)

			//assert that no error was thrown and expected value was returned
			g.Expect(err).Should(BeNil())
			g.Expect(APIServerFQDN).To(Equal(tt.APIServerFQDN))
		})
	}
}
