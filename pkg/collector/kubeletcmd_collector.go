package collector

import (
	"fmt"
	"runtime"

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

func (collector *KubeletCmdCollector) CheckSupported() error {
	// This looks to be impossible on Windows, since Windows containers don't support shared process namespaces,
	// and hence processes on the host are completely isolated from the container. See:
	// https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-containers/hyperv-container#piercing-the-isolation-boundary
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
	}

	return nil
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
