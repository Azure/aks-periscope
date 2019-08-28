package action

import (
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type kubeObjectsAction struct {
	name                     string
	collectIntervalInSeconds int
	processIntervalInSeconds int
	exportIntervalInSeconds  int
	exporter                 interfaces.Exporter
}

var _ interfaces.Action = &kubeObjectsAction{}

// NewKubeObjectsAction is a constructor
func NewKubeObjectsAction(collectIntervalInSeconds int, processIntervalInSeconds int, exportIntervalInSeconds int, exporter interfaces.Exporter) interfaces.Action {
	return &kubeObjectsAction{
		name:                     "kubeobjects",
		collectIntervalInSeconds: collectIntervalInSeconds,
		processIntervalInSeconds: processIntervalInSeconds,
		exportIntervalInSeconds:  exportIntervalInSeconds,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *kubeObjectsAction) GetName() string {
	return action.name
}

// Collect implements the interface method
func (action *kubeObjectsAction) Collect() ([]string, error) {
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
func (action *kubeObjectsAction) Process(collectFiles []string) ([]string, error) {
	return nil, nil
}

// Export implements the interface method
func (action *kubeObjectsAction) Export(exporter interfaces.Exporter, collectFiles []string, processfiles []string) error {
	if exporter != nil {
		return exporter.Export(append(collectFiles, processfiles...), action.exportIntervalInSeconds)
	}

	return nil
}
