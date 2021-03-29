package main

import (
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestEndToEndIntegrationSuccessCase(t *testing.T) {
	expectation := true

	output, err := utils.RunCommandOnContainer("kubectl", "apply", "-f", "../deployment/aks-periscope.yaml", "--validate=false")
	actual := successfulRun(t, output, err)

	if actual != expectation {
		t.Errorf("Expected successful return %v but got %v", expectation, actual)
	}
}

func TestEndToEndIntegrationUnsuccessFulCase(t *testing.T) {
	expectation := true

	output, err := utils.RunCommandOnContainer("kubectl", "apply", "-f", "../deployment/aks-periscope.yaml")
	actual := successfulRun(t, output, err)

	if actual != expectation {
		t.Logf("Expected successful return %v but got %v", expectation, actual)
	}
}

func successfulRun(t *testing.T, output interface{}, err interface{}) bool {
	if err != nil {
		t.Logf("unable to run periscope deployment file: %v\n", err)
		return false
	}

	t.Logf("successful output: %v\n", output)
	return true
}
