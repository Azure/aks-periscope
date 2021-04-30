package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestCustomResource(t *testing.T) {
	output, err := utils.RunCommandOnContainer("kubectl", "get", "namespace", "--output=jsonpath={.items..metadata.name}")
	if err != nil {
		t.Errorf("Error: %s", err)
	}
	namespaces := strings.Split(output, " ")
	for _, namespace := range namespaces {
		output, err = utils.RunCommandOnContainer("kubectl", "-n", namespace, "get", "crd", "--output=jsonpath={.items..metadata.name}")
		if err != nil {
			t.Errorf("Error: %s", err)
		}

		objects := strings.Split(output, " ")
		for _, object := range objects {
			output, err := utils.RunCommandOnContainer("kubectl", "-n", namespace, "describe", "crd", object)
			if err != nil {
				t.Errorf("Error: %s", err)
			} else {
				fmt.Printf(output)
			}
		}
	}
}
