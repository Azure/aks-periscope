package action

import (
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// KubeObjectsAction defines an action on describe kubernetes objects
type KubeObjectsAction struct{}

var _ interfaces.Action = &KubeObjectsAction{}

// GetName implements the interface method
func (action *KubeObjectsAction) GetName() string {
	return "kubeobjects"
}

// Collect implements the interface method
func (action *KubeObjectsAction) Collect() ([]string, error) {
	nameSpace := "kube-system"
	kubernetesObjects := []string{"pod", "service"}

	kubernetesObjectFiles := make([]string, 0)

	rootPath, _ := utils.CreateCollectorDir(action.GetName())

	for _, kubernetesObject := range kubernetesObjects {
		output, _ := utils.RunCommandOnHost("kubectl", "--kubeconfig", "/var/lib/kubelet/kubeconfig", "-n", nameSpace, "describe", kubernetesObject)

		kubernetesObjectFile := filepath.Join(rootPath, kubernetesObject)
		file, _ := os.Create(kubernetesObjectFile)
		defer file.Close()

		file.Write([]byte(output))

		kubernetesObjectFiles = append(kubernetesObjectFiles, kubernetesObjectFile)
	}

	return kubernetesObjectFiles, nil
}

// Process implements the interface method
func (action *KubeObjectsAction) Process([]string) error {
	return nil
}
