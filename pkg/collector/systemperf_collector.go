package collector

import (
	"github.com/Azure/aks-periscope/pkg/utils"
)

// SystemPerfCollector defines a SystemPerf Collector struct
type SystemPerfCollector struct {
	data map[string]string
}

// NewSystemPerfCollector is a constructor
func NewSystemPerfCollector() *SystemPerfCollector {
	return &SystemPerfCollector{
		data: make(map[string]string),
	}
}

func (collector *SystemPerfCollector) GetName() string {
	return "systemperf"
}

// Collect implements the interface method
func (collector *SystemPerfCollector) Collect() error {
	if err := utils.CreateKubeConfigFromServiceAccount(); err != nil {
		return err
	}

	output, err := utils.RunCommandOnContainer("kubectl", "top", "nodes")
	if err != nil {
		return err
	}

	collector.data["nodes"] = output

	output, err = utils.RunCommandOnContainer("kubectl", "top", "pods", "--all-namespaces")
	if err != nil {
		return err
	}

	collector.data["pods"] = output

	return nil
}

func (collector *SystemPerfCollector) GetData() map[string]string {
	return collector.data
}
