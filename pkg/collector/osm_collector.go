package collector

import (
	"log"
	"path/filepath"
	"regexp"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// OsmCollector defines an Osm Collector struct
type OsmCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &OsmCollector{}

// NewOsmCollector is a constructor
func NewOsmCollector(exporter interfaces.Exporter) *OsmCollector {
	return &OsmCollector{
		BaseCollector: BaseCollector{
			collectorType: Osm,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *OsmCollector) Collect() error {
	// Get all OSM deployments in order to collect information for various resources across all meshes in the cluster
	meshList, err := utils.GetResourceList([]string{"get", "deployments", "--all-namespaces", "-l", "app=osm-controller", "-o", "jsonpath={..meshName}"}, " ")
	if err != nil {
		return err
	}

	// Directory where OSM logs will be written to
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	for _, meshName := range meshList {
		meshRootPath := filepath.Join(rootPath, "mesh_"+meshName)

		monitoredNamespaces, err := utils.GetResourceList([]string{"get", "namespaces", "--all-namespaces", "-l", "openservicemesh.io/monitored-by=" + meshName, "-o", "jsonpath={..name}"}, " ")
		if err != nil {
			log.Printf("Failed to find any namespaces monitored by OSM named '%s': %+v\n", meshName, err)
		}
		controllerNamespaces, err := utils.GetResourceList([]string{"get", "deployments", "--all-namespaces", "-l", "app=osm-controller,meshName=" + meshName, "-o", "jsonpath={..metadata.namespace}"}, " ")
		if err != nil {
			log.Printf("Failed to find controller namespace(s) for OSM named '%s': %+v\n", meshName, err)
		}
		callNamespaceCollectors(collector, monitoredNamespaces, controllerNamespaces, meshRootPath, meshName)
		collectGroundTruth(collector, meshRootPath, meshName)
	}
	return nil
}

// callNamespaceCollectors calls functions to collect data for osm-controller namespace and namespaces monitored by a given mesh
func callNamespaceCollectors(collector *OsmCollector, monitoredNamespaces []string, controllerNamespaces []string, rootPath string, meshName string) {
	for _, namespace := range monitoredNamespaces {
		namespaceRootPath := filepath.Join(rootPath, "namespace_"+namespace)
		if err := collectDataFromEnvoys(collector, namespaceRootPath, namespace); err != nil {
			log.Printf("Failed to collect Envoy configs in OSM monitored namespace %s: %+v", namespace, err)
		}
		collectNamespaceResources(collector, namespaceRootPath, namespace)
	}
	for _, namespace := range controllerNamespaces {
		namespaceRootPath := filepath.Join(rootPath, "controller_namespace_"+namespace)
		if err := collectPodLogs(collector, namespaceRootPath, namespace); err != nil {
			log.Printf("Failed to collect pod logs for controller namespace %s: %+v", namespace, err)
		}
		collectNamespaceResources(collector, namespaceRootPath, namespace)
	}
}

// collectNamespaceResources collects information about general resources in a given namespace
func collectNamespaceResources(collector *OsmCollector, rootPath string, namespace string) {
	if err := collectPodConfigs(collector, rootPath, namespace); err != nil {
		log.Printf("Failed to collect pod configs for ns %s: %+v", namespace, err)
	}

	var namespaceResourcesMap = map[string][]string{
		"metadata.json":             {"get", "namespaces", namespace, "-o", "jsonpath={..metadata}", "-o", "json"},
		"services_list.tsv":         {"get", "services", "-n", namespace, "-o", "wide"},
		"services.json":             {"get", "services", "-n", namespace, "-o", "json"},
		"endpoints_list.tsv":        {"get", "endpoints", "-n", namespace, "-o", "wide"},
		"endpoints.json":            {"get", "endpoints", "-n", namespace, "-o", "json"},
		"configmaps_list.tsv":       {"get", "configmaps", "-n", namespace, "-o", "wide"},
		"configmaps.json":           {"get", "configmaps", "-n", namespace, "-o", "json"},
		"ingresses_list.tsv":        {"get", "ingresses", "-n", namespace, "-o", "wide"},
		"ingresses.json":            {"get", "ingresses", "-n", namespace, "-o", "json"},
		"service_accounts_list.tsv": {"get", "serviceaccounts", "-n", namespace, "-o", "wide"},
		"service_accounts.json":     {"get", "serviceaccounts", "-n", namespace, "-o", "json"},
		"pods_list.tsv":             {"get", "pods", "-n", namespace, "-o", "wide"},
	}
	for fileName, kubeCmds := range namespaceResourcesMap {
		if err := collector.CollectKubectlOutputToCollectorFiles(rootPath, fileName, kubeCmds); err != nil {
			log.Printf("Failed to collect %s in OSM monitored namespace %s: %+v", fileName, namespace, err)
		}
	}
}

// collectPodConfigs collects configs for pods in given namespace
func collectPodConfigs(collector *OsmCollector, rootPath string, namespace string) error {
	rootPath = filepath.Join(rootPath, "pod_configs")
	pods, err := utils.GetResourceList([]string{"get", "pods", "-n", namespace, "-o", "jsonpath={..metadata.name}"}, " ")
	if err != nil {
		return err
	}
	for _, podName := range pods {
		kubeCmds := []string{"get", "pods", "-n", namespace, podName, "-o", "json"}
		if err := collector.CollectKubectlOutputToCollectorFiles(rootPath, podName+".json", kubeCmds); err != nil {
			log.Printf("Failed to collect config for pod %s in OSM monitored namespace %s: %+v", podName, namespace, err)
		}
	}
	return nil
}

// collectDataFromEnvoys collects Envoy proxy config for pods in monitored namespace: port-forward and curl config dump
func collectDataFromEnvoys(collector *OsmCollector, rootPath string, namespace string) error {
	pods, err := utils.GetResourceList([]string{"get", "pods", "-n", namespace, "-o", "jsonpath={..metadata.name}"}, " ")
	if err != nil {
		return err
	}
	for _, podName := range pods {
		pid, err := utils.RunBackgroundCommand("kubectl", "port-forward", podName, "-n", namespace, "15000:15000")
		if err != nil {
			log.Printf("Failed to collect Envoy config for pod %s in OSM monitored namespace %s: %+v", podName, namespace, err)
			continue
		}

		envoyQueries := [5]string{"config_dump", "clusters", "listeners", "ready", "stats"}
		for _, query := range envoyQueries {
			responseBody, err := utils.GetUrlWithRetries("http://localhost:15000/"+query, 5)
			if err != nil {
				log.Printf("Failed to collect Envoy %s for pod %s in OSM monitored namespace %s: %+v", query, podName, namespace, err)
				continue
			}
			// Remove certificate secrets from Envoy config i.e., "inline_bytes" field from response
			re := regexp.MustCompile("(?m)[\r\n]+^.*inline_bytes.*$")
			secretRemovedResponse := re.ReplaceAllString(string(responseBody), "---redacted---")

			fileName := query + "_" + podName + ".txt"
			resourceFile := filepath.Join(rootPath, "envoy_data", fileName)
			if err = utils.WriteToFile(resourceFile, secretRemovedResponse); err != nil {
				log.Printf("Failed to write to file: %+v", err)
				continue
			}
			collector.AddToCollectorFiles(resourceFile)
		}
		if err = utils.KillProcess(pid); err != nil {
			log.Printf("Failed to kill process: %+v", err)
			continue
		}
	}
	return nil
}

// collectPodLogs collects logs of every pod in a given namespace
func collectPodLogs(collector *OsmCollector, rootPath string, namespace string) error {
	rootPath = filepath.Join(rootPath, "pod_logs")
	pods, err := utils.GetResourceList([]string{"get", "pods", "-n", namespace, "-o", "jsonpath={..metadata.name}"}, " ")
	if err != nil {
		return err
	}
	for _, podName := range pods {
		if err := collector.CollectKubectlOutputToCollectorFiles(rootPath, podName+".log", []string{"logs", "-n", namespace, podName}); err != nil {
			log.Printf("Failed to collect logs for pod %s: %+v", podName, err)
		}
	}
	return nil
}

// collectGroundTruth collects ground truth on resources in given mesh
func collectGroundTruth(collector *OsmCollector, rootPath string, meshName string) {
	var groundTruthMap = map[string][]string{
		"all_resources_list.tsv":                 {"get", "all", "--all-namespaces", "-l", "app.kubernetes.io/instance=" + meshName, "-o", "wide"},
		"all_resources_configs.json":             {"get", "all", "--all-namespaces", "-l", "app.kubernetes.io/instance=" + meshName, "-o", "json"},
		"mutating_webhook_configurations.json":   {"get", "MutatingWebhookConfiguration", "--all-namespaces", "-l", "app.kubernetes.io/instance=" + meshName, "-o", "json"},
		"validating_webhook_configurations.json": {"get", "ValidatingWebhookConfiguration", "--all-namespaces", "-l", "app.kubernetes.io/instance=" + meshName, "-o", "json"},
	}
	for fileName, kubeCmds := range groundTruthMap {
		if err := collector.CollectKubectlOutputToCollectorFiles(rootPath, fileName, kubeCmds); err != nil {
			log.Printf("Failed to collect %s for OSM: %+v", fileName, err)
		}
	}
}
