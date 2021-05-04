package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestExec(t *testing.T) {
	namespace := "azure-arc"
	output, err := utils.RunCommandOnContainer("kubectl", "get", "-n", namespace, "pods", "--output=jsonpath={.items..metadata.name}")
	if err != nil {
		t.Errorf("Error: %s", err)
	}
	pods := strings.Split(output, " ")
	for _, pod := range pods {
		output, err := utils.RunCommandOnContainer("kubectl", "-n", namespace, "exec", pod, "--", "curl", "https://partner.dp.kubernetesconfiguration-test.azure.com")
		if err != nil {
			if strings.Contains(err.Error(), "126") {
				fmt.Printf("Error!")
				continue
			}
			t.Errorf("Error: %s", err)
		}
		fmt.Printf(output)

	}
}
