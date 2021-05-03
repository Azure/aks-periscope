package collector

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// ExecCollector defines a Exec Collector struct
type ExecCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &ExecCollector{}

// ExecCollector is a constructor
func NewExecCollector(exporter interfaces.Exporter) *ExecCollector {
	return &ExecCollector{
		BaseCollector: BaseCollector{
			collectorType: Exec,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *ExecCollector) Collect() error {
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}
	namespaces := strings.Fields(os.Getenv("DIAGNOSTIC_EXEC_LIST"))
	for _, namespace := range namespaces {
		output, err := utils.RunCommandOnContainer("kubectl", "get", "-n", namespace, "pods", "--output=jsonpath={.items..metadata.name}")
		if err != nil {
			return err
		}
		pods := strings.Split(output, " ")

		for _, pod := range pods {
			execLog := filepath.Join(rootPath, namespace+"_"+pod)
			output, err := utils.RunCommandOnContainer("kubectl", "-n", namespace, "exec", pod, "--", "curl", "example.com")
			if err != nil {
				return err
			}
			err = utils.WriteToFile(execLog, output)
			if err != nil {
				return err
			}

			collector.AddToCollectorFiles(execLog)
		}
	}
	return nil
}
