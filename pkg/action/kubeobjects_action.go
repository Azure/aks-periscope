package action

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type kubeObjectsAction struct {
	name                     string
	collectIntervalInSeconds int
	collectCountForProcess   int
	collectCountForExport    int
	exporter                 interfaces.Exporter
	collectFiles             []string
	processFiles             []string
}

var _ interfaces.Action = &kubeObjectsAction{}

// NewKubeObjectsAction is a constructor
func NewKubeObjectsAction(collectIntervalInSeconds int, collectCountForProcess int, collectCountForExport int, exporter interfaces.Exporter) interfaces.Action {
	return &kubeObjectsAction{
		name:                     "kubeobjects",
		collectIntervalInSeconds: collectIntervalInSeconds,
		collectCountForProcess:   collectCountForProcess,
		collectCountForExport:    collectCountForExport,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *kubeObjectsAction) GetName() string {
	return action.name
}

// GetName implements the interface method
func (action *kubeObjectsAction) GetCollectIntervalInSeconds() int {
	return action.collectIntervalInSeconds
}

// GetName implements the interface method
func (action *kubeObjectsAction) GetCollectCountForProcess() int {
	return action.collectCountForProcess
}

// GetName implements the interface method
func (action *kubeObjectsAction) GetCollectCountForExport() int {
	return action.collectCountForExport
}

// Collect implements the interface method
func (action *kubeObjectsAction) Collect() error {
	action.collectFiles = []string{}

	nameSpaces := strings.Fields(os.Getenv("DIAGNOSTIC_KUBEOBJECTS_NAMESPACES"))
	kubernetesObjects := []string{"pod", "service"}
	rootPath, err := utils.CreateCollectorDir(action.GetName())
	if err != nil {
		return err
	}

	for _, nameSpace := range nameSpaces {
		err := os.MkdirAll(filepath.Join(rootPath, nameSpace), os.ModePerm)
		if err != nil {
			return fmt.Errorf("Fail to create dir %s: %+v", filepath.Join(rootPath, nameSpace), err)
		}

		for _, kubernetesObject := range kubernetesObjects {
			kubernetesObjectFile := filepath.Join(rootPath, nameSpace, kubernetesObject)

			output, err := utils.RunCommandOnHost("kubectl", "--kubeconfig", "/var/lib/kubelet/kubeconfig", "-n", nameSpace, "describe", kubernetesObject)
			if err != nil {
				return err
			}

			err = utils.WriteToFile(kubernetesObjectFile, output)
			if err != nil {
				return err
			}

			action.collectFiles = append(action.collectFiles, kubernetesObjectFile)
		}
	}

	return nil
}

// Process implements the interface method
func (action *kubeObjectsAction) Process() error {
	return nil
}

// Export implements the interface method
func (action *kubeObjectsAction) Export() error {
	if action.exporter != nil {
		return action.exporter.Export(append(action.collectFiles, action.processFiles...))
	}

	return nil
}
