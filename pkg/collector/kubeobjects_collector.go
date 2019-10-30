package collector

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// KubeObjectsCollector defines a KubeObjects Collector struct
type KubeObjectsCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &KubeObjectsCollector{}

// NewKubeObjectsCollector is a constructor
func NewKubeObjectsCollector(exporter interfaces.Exporter) *KubeObjectsCollector {
	return &KubeObjectsCollector{
		BaseCollector: BaseCollector{
			collectorType: KubeObjects,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *KubeObjectsCollector) Collect() error {
	kubernetesObjects := strings.Fields(os.Getenv("DIAGNOSTIC_KUBEOBJECTS_LIST"))
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
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

			collector.AddToCollectorFiles(kubernetesObjectFile)
		}
	}

	return nil
}
