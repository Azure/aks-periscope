package collector

import (
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// CustomResourceCollector defines a CustomResources Collector struct
type CustomResourceCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &CustomResourceCollector{}

// CustomResourceCollector is a constructor
func NewCustomResourceCollector(exporter interfaces.Exporter) *CustomResourceCollector {
	return &CustomResourceCollector{
		BaseCollector: BaseCollector{
			collectorType: CustomResource,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *CustomResourceCollector) Collect() error {
	rootPath, err := utils.CreateCollectorDir(collector.GetName())

	output, err := utils.RunCommandOnContainer("kubectl", "get", "namespace", "--output=jsonpath={.items..metadata.name}")
	if err != nil {
		return err
	}
	namespaces := strings.Split(output, " ")
	for _, namespace := range namespaces {
		output, err = utils.RunCommandOnContainer("kubectl", "-n", namespace, "get", "crd", "--output=jsonpath={.items..metadata.name}")
		if err != nil {
			return err
		}

		objects := strings.Split(output, " ")
		for _, object := range objects {
			customResourceFile := filepath.Join(rootPath, namespace+"_"+"crd"+"_"+object)
			output, err := utils.RunCommandOnContainer("kubectl", "-n", namespace, "describe", "crd", object)
			if err != nil {
				return err
			}
			err = utils.WriteToFile(customResourceFile, output)
			if err != nil {
				return err
			}

			collector.AddToCollectorFiles(customResourceFile)
		}
	}
	return nil
}
