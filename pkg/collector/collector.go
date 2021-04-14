package collector

import (
	"path/filepath"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// Type defines Collector Type
type Type int

const (
	// DNS defines DNS Collector Type
	DNS Type = iota
	// ContainerLogs defines ContainerLogs Collector Type
	ContainerLogs
	//Helm defines Helm Collector Type
	Helm
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
	// Osm defines Open Service Mesh Collector Type
	Osm
	// SystemLogs defines SystemLogs Collector Type
	SystemLogs
	// SystemPerf defines SystemPerf Collector Type
	SystemPerf
)

// Name returns type name
func (t Type) name() string {
	return [...]string{"dns", "containerlogs", "helm", "iptables", "kubeletcmd", "kubeobjects", "networkoutbound", "nodelogs", "osm", "systemlogs", "systemperf"}[t]
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

// CollectKubectlOutputToCollectorFiles collects output of a given kubectl command to a file.
// Returns kubectl's stderr output if stdout output is empty.
func (b *BaseCollector) CollectKubectlOutputToCollectorFiles(rootPath string, fileName string, kubeCmds []string) error {
	outputStreams, err := utils.RunCommandOnContainerWithOutputStreams("kubectl", kubeCmds...)
	if err != nil {
		return err
	}

	// If kubectl stdout output is empty, i.e., there is no resource of this type within the cluster
	// the absence of this resource is logged in the file with the relevant message from stderr (Ex: "No resource found...").
	output := outputStreams.Stdout
	if len(output) == 0 {
		output = outputStreams.Stderr
	}

	resourceFile := filepath.Join(rootPath, fileName)
	if err = utils.WriteToFile(resourceFile, output); err != nil {
		return err
	}

	b.AddToCollectorFiles(resourceFile)

	return nil
}
