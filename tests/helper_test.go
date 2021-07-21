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
