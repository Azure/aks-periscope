package collector

import (
	"path/filepath"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// OsmLogsCollector defines an OsmLogs Collector struct
type OsmLogsCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &OsmLogsCollector{}

// NewOsmLogsCollector is a constructor
func NewOsmLogsCollector(exporter interfaces.Exporter) *OsmLogsCollector {
	return &OsmLogsCollector{
		BaseCollector: BaseCollector{
			collectorType: OsmLogs,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *OsmLogsCollector) Collect() error {
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}
	// can define resources to query in deployment.yaml and iterate through the commands+resources needed and create multiple files
	// see kubeobject collector for example

	// Get osm resources as table
	allResourcesFile := filepath.Join(rootPath, "allResources")
	output, err := utils.RunCommandOnContainer("kubectl", "get", "all", "--all-namespaces", "--selector", "app.kubernetes.io/name=openservicemesh.io")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(allResourcesFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(allResourcesFile)

	// Get osm resource configs
	allResourceConfigsFile := filepath.Join(rootPath, "allResourceConfigs")
	output, err = utils.RunCommandOnContainer("kubectl", "get", "all", "--all-namespaces", "--selector", "app.kubernetes.io/name=openservicemesh.io", "-o", "json")
	if err != nil {
		return err
	}

	err = utils.WriteToFile(allResourceConfigsFile, output)
	if err != nil {
		return err
	}

	collector.AddToCollectorFiles(allResourceConfigsFile)

	return nil
}

// Helper function to get all meshes in the cluster
func getMeshList() (string, error) {
	meshList, err := utils.RunCommandOnContainer("kubectl", "get", "deployments", "--all-namespaces", "-o=jsonpath=\"{..meshName}\"", "-l", "app=osm-controller")
	if err != nil {
		return "", err
	}
	return meshList, nil
}
