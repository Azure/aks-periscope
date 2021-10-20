package collector

import (
	"fmt"
	"log"
	"regexp"

	"github.com/Azure/aks-periscope/pkg/utils"
)

// OsmCollector defines an OSM Collector struct
type OsmCollector struct {
	data map[string]string
}

// NewOsmCollector is a constructor
func NewOsmCollector() *OsmCollector {
	return &OsmCollector{
		data: make(map[string]string),
	}
}

func (collector *OsmCollector) GetName() string {
	return "osm"
}

// Collect implements the interface method
func (collector *OsmCollector) Collect() error {
	// Get all OSM deployments in order to collect information for various resources across all meshes in the cluster
	meshList, err := utils.GetResourceList([]string{"get", "deployments", "--all-namespaces", "-l", "app=osm-controller", "-o", "jsonpath={..meshName}"}, " ")
	if err != nil {
		return err
	}

	for _, meshName := range meshList {
		monitoredNamespaces, err := utils.GetResourceList([]string{"get", "namespaces", "--all-namespaces", "-l", "openservicemesh.io/monitored-by=" + meshName, "-o", "jsonpath={..name}"}, " ")
		if err != nil {
			log.Printf("Failed to find any namespaces monitored by OSM named '%s': %+v\n", meshName, err)
		}
		controllerNamespaces, err := utils.GetResourceList([]string{"get", "deployments", "--all-namespaces", "-l", "app=osm-controller,meshName=" + meshName, "-o", "jsonpath={..metadata.namespace}"}, " ")
		if err != nil {
			log.Printf("Failed to find controller namespace(s) for OSM named '%s': %+v\n", meshName, err)
		}
		collector.callNamespaceCollectors(monitoredNamespaces, controllerNamespaces, meshName)
		collector.collectGroundTruth(meshName)
	}
	return nil
}

// callNamespaceCollectors calls functions to collect data for osm-controller namespace and namespaces monitored by a given mesh
func (collector *OsmCollector) callNamespaceCollectors(monitoredNamespaces []string, controllerNamespaces []string, meshName string) {
	for _, namespace := range monitoredNamespaces {
		if err := collector.collectDataFromEnvoys(namespace, meshName); err != nil {
			log.Printf("Failed to collect Envoy configs in OSM monitored namespace %s: %+v", namespace, err)
		}
		collector.collectNamespaceResources(namespace, meshName)
	}
	for _, namespace := range controllerNamespaces {
		if err := collector.collectPodLogs(namespace, meshName); err != nil {
			log.Printf("Failed to collect pod logs for controller namespace %s: %+v", namespace, err)
		}
		collector.collectNamespaceResources(namespace, meshName)
	}
}

// collectNamespaceResources collects information about general resources in a given namespace
func (collector *OsmCollector) collectNamespaceResources(namespace string, meshName string) {
	if err := collector.collectPodConfigs(namespace, meshName); err != nil {
		log.Printf("Failed to collect pod configs for ns %s: %+v", namespace, err)
	}

	metadata, err := utils.RunCommandOnContainer("kubectl", "get", "namespaces", namespace, "-o", "jsonpath={..metadata}", "-o", "json")
	if err != nil {
		metadata = fmt.Sprintf("Failed to collect metadata for namespace %s: %v", namespace, err)
		log.Print(metadata)
	}

	servicesList, err := utils.RunCommandOnContainer("kubectl", "get", "services", "-n", namespace, "-o", "wide")
	if err != nil {
		servicesList = fmt.Sprintf("Failed to collect services for namespace %s: %v", namespace, err)
		log.Print(servicesList)
	}

	services, err := utils.RunCommandOnContainer("kubectl", "get", "services", "-n", namespace, "-o", "json")
	if err != nil {
		services = fmt.Sprintf("Failed to collect services for namespace %s: %v", namespace, err)
		log.Print(services)
	}

	endpointList, err := utils.RunCommandOnContainer("kubectl", "get", "endpoints", "-n", namespace, "-o", "wide")
	if err != nil {
		endpointList = fmt.Sprintf("Failed to collect endpoints for namespace %s: %v", namespace, err)
		log.Print(endpointList)
	}

	endpoints, err := utils.RunCommandOnContainer("kubectl", "get", "endpoints", "-n", namespace, "-o", "json")
	if err != nil {
		endpoints = fmt.Sprintf("Failed to collect endpoints for namespace %s: %v", namespace, err)
		log.Print(endpoints)
	}

	configmapsList, err := utils.RunCommandOnContainer("kubectl", "get", "configmaps", "-n", namespace, "-o", "wide")
	if err != nil {
		configmapsList = fmt.Sprintf("Failed to collect configmaps for namespace %s: %v", namespace, err)
		log.Print(configmapsList)
	}

	configmaps, err := utils.RunCommandOnContainer("kubectl", "get", "configmaps", "-n", namespace, "-o", "json")
	if err != nil {
		configmaps = fmt.Sprintf("Failed to collect configmaps for namespace %s: %v", namespace, err)
		log.Print(configmaps)
	}

	ingressList, err := utils.RunCommandOnContainer("kubectl", "get", "ingresses", "-n", namespace, "-o", "wide")
	if err != nil {
		ingressList = fmt.Sprintf("Failed to collect ingresses for namespace %s: %v", namespace, err)
		log.Print(ingressList)
	}

	ingresses, err := utils.RunCommandOnContainer("kubectl", "get", "ingresses", "-n", namespace, "-o", "json")
	if err != nil {
		ingresses = fmt.Sprintf("Failed to collect ingresses for namespace %s: %v", namespace, err)
		log.Print(ingresses)
	}

	svcAccountList, err := utils.RunCommandOnContainer("kubectl", "get", "serviceaccounts", "-n", namespace, "-o", "wide")
	if err != nil {
		svcAccountList = fmt.Sprintf("Failed to collect service accounts for namespace %s: %v", namespace, err)
		log.Print(svcAccountList)
	}

	svcAccounts, err := utils.RunCommandOnContainer("kubectl", "get", "serviceaccounts", "-n", namespace, "-o", "json")
	if err != nil {
		svcAccounts = fmt.Sprintf("Failed to collect service accounts for namespace %s: %v", namespace, err)
		log.Print(svcAccounts)
	}

	podList, err := utils.RunCommandOnContainer("kubectl", "get", "pods", "-n", namespace, "-o", "wide")
	if err != nil {
		podList = fmt.Sprintf("Failed to collect pod list for namespace %s: %v", namespace, err)
		log.Print(podList)
	}
	filePath := meshName + "/" + namespace
	collector.data[filePath+"_metadata"] = metadata
	collector.data[filePath+"_services_list"] = servicesList
	collector.data[filePath+"_services"] = services
	collector.data[filePath+"_endpoints_list"] = endpointList
	collector.data[filePath+"_endpoints"] = endpoints
	collector.data[filePath+"_configmaps_list"] = configmapsList
	collector.data[filePath+"_configmaps"] = configmaps
	collector.data[filePath+"_ingresses_list"] = ingressList
	collector.data[filePath+"_ingresses"] = ingresses
	collector.data[filePath+"_service_accounts_list"] = svcAccountList
	collector.data[filePath+"_service_accounts"] = svcAccounts
	collector.data[filePath+"_pods_list"] = podList
}

// collectPodConfigs collects configs for pods in given namespace
func (collector *OsmCollector) collectPodConfigs(namespace string, meshName string) error {
	pods, err := utils.GetResourceList([]string{"get", "pods", "-n", namespace, "-o", "jsonpath={..metadata.name}"}, " ")
	if err != nil {
		return err
	}

	for _, podName := range pods {
		output, err := utils.RunCommandOnContainer("kubectl", "get", "pods", "-n", namespace, podName, "-o", "json")
		if err != nil {
			output = fmt.Sprintf("Failed to collect config for pod %s in OSM monitored namespace %s: %v", podName, namespace, err)
			log.Print(output)
		}
		filePath := meshName + "/" + podName + "_podConfig"
		collector.data[filePath] = output
	}

	return nil
}

// collectDataFromEnvoys collects Envoy proxy config for pods in monitored namespace: port-forward and curl config dump
func (collector *OsmCollector) collectDataFromEnvoys(namespace string, meshName string) error {
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
			filePath := meshName + "/envoy/"+ podName + query
			collector.data[filePath] = secretRemovedResponse
		}
		if err = utils.KillProcess(pid); err != nil {
			log.Printf("Failed to kill process: %+v", err)
			continue
		}
	}
	return nil
}

// collectPodLogs collects logs of every pod in a given namespace
func (collector *OsmCollector) collectPodLogs(namespace string, meshName string) error {
	pods, err := utils.GetResourceList([]string{"get", "pods", "-n", namespace, "-o", "jsonpath={..metadata.name}"}, " ")
	if err != nil {
		return err
	}
	for _, podName := range pods {
		output, err := utils.RunCommandOnContainer("kubectl", "logs", "-n", namespace, podName)
		if err != nil {
			output = fmt.Sprintf("Failed to collect logs for pod %s: %+v", podName, err)
			log.Print(output)
		}
		filePath := meshName + "/" + podName + "_podLogs"
		collector.data[filePath] = output
	}
	return nil
}

// collectGroundTruth collects ground truth on resources in given mesh
func (collector *OsmCollector) collectGroundTruth(meshName string) {
	allResourcesList, err := utils.RunCommandOnContainer("kubectl", "get", "all", "--all-namespaces", "-l", "app.kubernetes.io/instance="+meshName, "-o", "wide")
	if err != nil {
		allResourcesList = fmt.Sprintf("Failed to collect all resources list for mesh %s: %v", meshName, err)
		log.Print(allResourcesList)
	}

	allResourcesConfigs, err := utils.RunCommandOnContainer("kubectl", "get", "all", "--all-namespaces", "-l", "app.kubernetes.io/instance="+meshName, "-o", "json")
	if err != nil {
		allResourcesConfigs = fmt.Sprintf("Failed to collect all resources configs for mesh %s: %v", meshName, err)
		log.Print(allResourcesConfigs)
	}

	mutationWebhookConfig, err := utils.RunCommandOnContainer("kubectl", "get", "MutatingWebhookConfiguration", "--all-namespaces", "-l", "app.kubernetes.io/instance="+meshName, "-o", "json")
	if err != nil {
		mutationWebhookConfig = fmt.Sprintf("Failed to collect mutating webhook config for mesh %s: %v", meshName, err)
		log.Print(mutationWebhookConfig)
	}

	validatingWebhookConfig, err := utils.RunCommandOnContainer("kubectl", "get", "ValidatingWebhookConfiguration", "--all-namespaces", "-l", "app.kubernetes.io/instance="+meshName, "-o", "json")
	if err != nil {
		validatingWebhookConfig = fmt.Sprintf("Failed to collect validating webhook config for mesh %s: %v", meshName, err)
		log.Print(validatingWebhookConfig)
	}

	meshConfig, err := utils.RunCommandOnContainer("kubectl", "get", "meshconfigs", "--all-namespaces", "-o", "json")
	if err != nil {
		meshConfig = fmt.Sprintf("Failed to collect meshconfigs for mesh %s: %v", meshName, err)
		log.Print(meshConfig)
	}
	filePath := meshName + "/control_plane"
	collector.data[filePath+"/all_resources_list"] = allResourcesList
	collector.data[filePath+"/all_resources_configs"] = allResourcesConfigs
	collector.data[filePath+"/mutating_webhook_configurations"] = mutationWebhookConfig
	collector.data[filePath+"/validating_webhook_configurations"] = validatingWebhookConfig
	collector.data[filePath+"/mesh_configs"] = meshConfig
}

func (collector *OsmCollector) GetData() map[string]string {
	return collector.data
}
