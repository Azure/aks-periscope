package action

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
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

	kubernetesObjects := strings.Fields(os.Getenv("DIAGNOSTIC_KUBEOBJECTS_LIST"))
	rootPath, err := utils.CreateCollectorDir(action.GetName())
	if err != nil {
		return err
	}

	for _, kubernetesObject := range kubernetesObjects {
		kubernetesObjectParts := strings.Split(kubernetesObject, "/")
		nameSpace := kubernetesObjectParts[0]
		objectType := kubernetesObjectParts[1]
		objects := []string{}
		if len(kubernetesObjectParts) == 3 {
			objects = append(objects, kubernetesObjectParts[2])
		}

		if len(objects) == 0 {
			output, err := utils.RunCommandOnContainer("kubectl", "-n", nameSpace, "get", objectType, "--output=jsonpath={.items..metadata.name}")
			if err != nil {
				return err
			}

			objects = strings.Split(output, " ")
		}

		for _, object := range objects {
			kubernetesObjectFile := filepath.Join(rootPath, nameSpace+"_"+objectType+"_"+object)

			output, err := utils.RunCommandOnContainer("kubectl", "-n", nameSpace, "describe", objectType, object)
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
