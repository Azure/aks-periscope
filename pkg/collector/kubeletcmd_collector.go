package collector

import (
	"github.com/Azure/aks-periscope/pkg/utils"
)

// KubeletCmdCollector defines a KubeletCmd Collector struct
type KubeletCmdCollector struct {
	data map[string]string
}

// NewKubeletCmdCollector is a constructor
func NewKubeletCmdCollector() *KubeletCmdCollector {
	return &KubeletCmdCollector{
		data: make(map[string]string),
	}
}

func (collector *KubeletCmdCollector) GetName() string {
	return "kubeletcmd"
}

// Collect implements the interface method
func (collector *KubeletCmdCollector) Collect() error {
	output, err := utils.RunCommandOnHost("ps", "-o", "cmd=", "-C", "kubelet")
	if err != nil {
		return err
	}

	collector.data["kubeletcmd"] = output

	return nil
}

func (collector *KubeletCmdCollector) GetData() map[string]string {
	return collector.data
}
