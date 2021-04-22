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

	// * Collect information for various resources across all meshes in the cluster
	meshList, err := getResourceList("deployments", "app=osm-controller", "-o=jsonpath={..meshName}")
	if err != nil {
		return err
	}

	for _, meshName := range meshList {
		namespaceInMesh, err := getResourceList("namespaces", "openservicemesh.io/monitored-by="+meshName, "-o=jsonpath={..name}")
		if err != nil {
			return err
		}
		collectNamespaceMetadata(collector, namespaceInMesh, rootPath, meshName)
	}

	return nil
}

// * Collects metadata for each ns in a given mesh
func collectNamespaceMetadata(collector *OsmLogsCollector, namespaces []string, rootPath, meshName string) error {
	for _, namespace := range namespaces {
		namespaceMetadataFile := filepath.Join(rootPath, meshName+"_"+namespace+"_"+"metadata")
		namespaceMetadata, err := utils.RunCommandOnContainer("kubectl", "get", "namespaces", namespace, "-o=jsonpath={..metadata}", "-o", "json")
		if err != nil {
			return err
		}
		err = utils.WriteToFile(namespaceMetadataFile, namespaceMetadata)
		if err != nil {
			return err
		}
		collector.AddToCollectorFiles(namespaceMetadataFile)
	}

	return nil
}

// Helper function to get all meshes in the cluster
func getResourceList(resource, label, outputFormat string) ([]string, error) {
	resourceList, err := utils.RunCommandOnContainer("kubectl", "get", resource, "--all-namespaces", "--selector", label, outputFormat)
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.Trim(resourceList, "\""), " "), nil
}
