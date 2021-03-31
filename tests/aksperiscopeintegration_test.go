package main

import (
	"fmt"
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
	. "github.com/onsi/gomega"
)

func TestEndToEndIntegrationSuccessCase(t *testing.T) {
	runperiscopedeploycommand(t, false)
}

func TestEndToEndIntegrationUnsuccessFulCase(t *testing.T) {
	runperiscopedeploycommand(t, true)
}

func runperiscopedeploycommand(t *testing.T, validate bool) {
	// This flag switch on and off for storage account validation.
	validateflag := fmt.Sprintf("--validate=%v", validate)
	g := NewGomegaWithT(t)

	output, err := utils.RunCommandOnContainer("kubectl", "apply", "-f", "../deployment/aks-periscope.yaml", validateflag)

	if err != nil && validate {
		g.Expect(err).Should(HaveOccurred())
		t.Logf("unsuccessful output: %v\n", err)
	}

	if output != "" && !validate {
		g.Expect(err).ToNot(HaveOccurred())
		t.Logf("successful output: %v\n", output)
	}
}
