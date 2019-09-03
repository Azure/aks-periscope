package action

import (
	"path/filepath"
	"time"

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
	rootPath, _ := utils.CreateCollectorDir(action.GetName())

	kubernetesObjectFiles := make([]string, 0)
	for _, kubernetesObject := range kubernetesObjects {
		kubernetesObjectFile := filepath.Join(rootPath, kubernetesObject)

		go func(nameSpace string, kubernetesObject string, kubernetesObjectFile string) {
			ticker := time.NewTicker(time.Duration(action.collectIntervalInSeconds) * time.Second)
			for {
				select {
				case <-ticker.C:
					collectKubeObjects(nameSpace, kubernetesObject, kubernetesObjectFile)
				}
			}
		}(nameSpace, kubernetesObject, kubernetesObjectFile)

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

func collectKubeObjects(nameSpace string, kubernetesObject string, kubernetesObjectFile string) error {
	output, _ := utils.RunCommandOnHost("kubectl", "--kubeconfig", "/var/lib/kubelet/kubeconfig", "-n", nameSpace, "describe", kubernetesObject)
	err := utils.WriteToFile(kubernetesObjectFile, output)
	if err != nil {
		return err
	}

	return nil
}
