package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
	. "github.com/onsi/gomega"
)

func TestEndToEndIntegrationSuccessCase(t *testing.T) {
	runperiscopedeploycommand(t, false)
	g := NewGomegaWithT(t)

	g.Eventually(func() bool {
		return checkifpodsrunning(t)
	}, "60s", "2s").Should(BeTrue())

	/*
			More integration test expansiont Ideas:
			Further work: ToExpand:
		 		* Another good integration test could be the lcoation the logs get collected before zipping and prior to storage call.
				* /var/logs/aks-periscope/* - is the location I think.
				* funct : zipAndExport has more details.
	*/

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

func checkifpodsrunning(t *testing.T) bool {
	g := NewGomegaWithT(t)

	output, err := utils.RunCommandOnContainer("kubectl", "get", "pods", "-n", "aks-periscope")
	firstpod := strings.Split(output, "\n")

	firstpodname := strings.Fields(firstpod[1])[0]
	firstpodstate := strings.Fields(firstpod[1])[2]

	t.Logf(" Outcome is %v ===> %v", firstpodname, firstpodstate)

	if err != nil {
		g.Expect(err).ToNot(HaveOccurred())
		t.Logf("unsuccessful error: %v\n", err)
	}

	if output != "" {
		g.Expect(err).ToNot(HaveOccurred())
		t.Logf("successful output: %v\n", output)
	}

	if firstpodstate == "Running" {
		return true
	}

	return false
}
