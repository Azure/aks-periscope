package collector

import (
	"os"
	"strings"

	"github.com/Azure/aks-periscope/pkg/utils"
)

// KubeObjectsCollector defines a KubeObjects Collector struct
type KubeObjectsCollector struct {
	data map[string]string
}

// NewKubeObjectsCollector is a constructor
func NewKubeObjectsCollector() *KubeObjectsCollector {
	return &KubeObjectsCollector{
		data: make(map[string]string),
	}
}

func (collector *KubeObjectsCollector) GetName() string {
	return "kubeobjects"
}

// Collect implements the interface method
func (collector *KubeObjectsCollector) Collect() error {
	kubernetesObjects := strings.Fields(os.Getenv("DIAGNOSTIC_KUBEOBJECTS_LIST"))

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

			output, err := utils.RunCommandOnContainer("kubectl", "-n", nameSpace, "describe", objectType, object)
			if err != nil {
				return err
			}

			collector.data[nameSpace+"_"+objectType+"_"+object] = output
		}
	}

	return nil
}

func (collector *KubeObjectsCollector) GetData() map[string]string {
	return collector.data
}
