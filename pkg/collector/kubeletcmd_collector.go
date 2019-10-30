package collector

import (
	"path/filepath"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// KubeletCmdCollector defines a KubeletCmd Collector struct
type KubeletCmdCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &KubeletCmdCollector{}

// NewKubeletCmdCollector is a constructor
func NewKubeletCmdCollector(exporter interfaces.Exporter) *KubeletCmdCollector {
	return &KubeletCmdCollector{
		BaseCollector: BaseCollector{
			collectorType: KubeletCmd,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *KubeletCmdCollector) Collect() error {
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	kubeletcmdFile := filepath.Join(rootPath, collector.GetName())

	output, err := utils.RunCommandOnHost("ps", "-o", "cmd=", "-C", "kubelet")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(kubeletcmdFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(kubeletcmdFile)

	return nil
}
