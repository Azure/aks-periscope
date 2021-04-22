package tests

import (
	"fmt"
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestHelm(t *testing.T) {

	output, err := utils.RunCommandOnContainer("helm", "list", "--all-namespaces")
	if err != nil {
		t.Errorf("Error: %s", err)
	} else {
		fmt.Printf(output)
	}
	output, err = utils.RunCommandOnContainer("helm", "history", "-n", "default", "azure-arc")
	if err != nil {
		t.Errorf("Error: %s", err)
	} else {
		fmt.Printf(output)
	}
}
