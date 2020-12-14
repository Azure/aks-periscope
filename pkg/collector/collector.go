package collector

import (
	"github.com/Azure/aks-periscope/pkg/interfaces"
)

// Type defines Collector Type
type Type int

const (
	// DNS defines DNS Collector Type
	DNS Type = iota
	// ContainerLogs defines ContainerLogs Collector Type
	ContainerLogs
	// IPTables defines IPTables Collector Type
	IPTables
	// KubeletCmd defines KubeletCmd Collector Type
	KubeletCmd
	// KubeObjects defines KubeObjects Collector Type
	KubeObjects
	// NetworkOutbound defines NetworkOutbound Collector Type
	NetworkOutbound
	// NodeLogs defines NodeLogs Collector Type
	NodeLogs
	// SystemLogs defines SystemLogs Collector Type
	SystemLogs
	// SystemPerf defines SystemPerf Collector Type
	SystemPerf
)

// Name returns type name
func (t Type) name() string {
	return [...]string{"dns", "containerlogs", "iptables", "kubeletcmd", "kubeobjects", "networkoutbound", "nodelogs", "systemlogs", "systemperf"}[t]
}

// BaseCollector defines Base Collector
type BaseCollector struct {
	collectorType            Type
	collectIntervalInSeconds int
	collectorFiles           []string
	exporter                 interfaces.Exporter
}

// GetName gets collector name
func (b *BaseCollector) GetName() string {
	return b.collectorType.name()
}

// GetCollectIntervalInSeconds gets collector interval in seconds
func (b *BaseCollector) GetCollectIntervalInSeconds() int {
	return b.collectIntervalInSeconds
}

// GetCollectorFiles gets collector files
func (b *BaseCollector) GetCollectorFiles() []string {
	return b.collectorFiles
}

// AddToCollectorFiles adds a file to collector files
func (b *BaseCollector) AddToCollectorFiles(file string) {
	b.collectorFiles = append(b.collectorFiles, file)
}

// Export implements the interface method
func (b *BaseCollector) Export() error {
	if b.exporter != nil {
		return b.exporter.Export(b.collectorFiles)
	}

	return nil
}
