package collector

import (
	"path/filepath"
	"strings"

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

	// * Get osm resources as table
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

	// * Get osm resource configs
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

	// * Get metadata of all namespaces in all OSMs
	meshList, err := getMeshList()
	if err != nil {
		return err
	}

	for _, meshName := range meshList {
		meshName = strings.Trim(meshName, "\"")
		namespacesInMesh, err := utils.RunCommandOnContainer("kubectl", "get", "namespaces", "--selector", "openservicemesh.io/monitored-by="+meshName, "-o=jsonpath={..name}")
		if err != nil {
			return err
		}

		// Create a metadata file for each namespace in that mesh
		for _, namespace := range strings.Split(namespacesInMesh, " ") {
			namespace = strings.Trim(namespace, "\"")
			namespaceMetadataFile := filepath.Join(rootPath, meshName+"_"+namespace+"_"+"metadata")
			namespaceMetadata, err := utils.RunCommandOnContainer("kubectl", "get", "namespaces", namespace, "-o=jsonpath={..metadata}")
			if err != nil {
				return err
			}
			err = utils.WriteToFile(namespaceMetadataFile, namespaceMetadata)
			if err != nil {
				return err
			}
			collector.AddToCollectorFiles(namespaceMetadataFile)
		}
	}

	return nil
}

// Helper function to get all meshes in the cluster
func getMeshList() ([]string, error) {
	meshList, err := utils.RunCommandOnContainer("kubectl", "get", "deployments", "--all-namespaces", "--selector", "app=osm-controller", "-o=jsonpath={..meshName}")
	if err != nil {
		return nil, err
	}
	return strings.Split(meshList, " "), nil
}
