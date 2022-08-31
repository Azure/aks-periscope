package collector

import (
	"fmt"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// KubeletCmdCollector defines a KubeletCmd Collector struct
type KubeletCmdCollector struct {
	KubeletCommand string
	osIdentifier   utils.OSIdentifier
	runtimeInfo    *utils.RuntimeInfo
}

// NewKubeletCmdCollector is a constructor
func NewKubeletCmdCollector(osIdentifier utils.OSIdentifier, runtimeInfo *utils.RuntimeInfo) *KubeletCmdCollector {
	return &KubeletCmdCollector{
		KubeletCommand: "",
		osIdentifier:   osIdentifier,
		runtimeInfo:    runtimeInfo,
	}
}

func (collector *KubeletCmdCollector) GetName() string {
	return "kubeletcmd"
}

func (collector *KubeletCmdCollector) CheckSupported() error {
	// This looks to be impossible on Windows, since Windows containers don't support shared process namespaces,
	// and hence processes on the host are completely isolated from the container. See:
	// https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-containers/hyperv-container#piercing-the-isolation-boundary
	if collector.osIdentifier != utils.Linux {
		return fmt.Errorf("unsupported OS: %s", collector.osIdentifier)
	}

	if utils.Contains(collector.runtimeInfo.CollectorList, "connectedCluster") {
		return fmt.Errorf("not included because 'connectedCluster' is in COLLECTOR_LIST variable. Included values: %s", strings.Join(collector.runtimeInfo.CollectorList, " "))
	}

	return nil
}

// Collect implements the interface method
func (collector *KubeletCmdCollector) Collect() error {
	output, err := utils.RunCommandOnHost("ps", "-o", "cmd=", "-C", "kubelet")
	if err != nil {
		return err
	}

	collector.KubeletCommand = output

	return nil
}

func (collector *KubeletCmdCollector) GetData() map[string]interfaces.DataValue {
	return map[string]interfaces.DataValue{
		"kubeletcmd": utils.NewStringDataValue(collector.KubeletCommand),
	}
}
