package collector

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Azure/aks-periscope/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// OsmCollector defines an OSM Collector struct
type OsmCollector struct {
	data          map[string]string
	kubeconfig    *rest.Config
	commandRunner *utils.KubeCommandRunner
	runtimeInfo   *utils.RuntimeInfo
}

// NewOsmCollector is a constructor
func NewOsmCollector(config *rest.Config, runtimeInfo *utils.RuntimeInfo) *OsmCollector {
	return &OsmCollector{
		data:          make(map[string]string),
		kubeconfig:    config,
		commandRunner: utils.NewKubeCommandRunner(config),
		runtimeInfo:   runtimeInfo,
	}
}

func (collector *OsmCollector) GetName() string {
	return "osm"
}

func (collector *OsmCollector) CheckSupported() error {
	// This is not currently supported on Windows because it launches `kubectl` as a separate process (within GetResourceList).
	// If/when it is reimplemented using the go client API for k8s, we can re-enable this.
	if collector.runtimeInfo.OSIdentifier != "linux" {
		return fmt.Errorf("unsupported OS: %s", collector.runtimeInfo.OSIdentifier)
	}

	if !utils.Contains(collector.runtimeInfo.CollectorList, "OSM") {
		return fmt.Errorf("not included because 'OSM' not in COLLECTOR_LIST variable. Included values: %s", strings.Join(collector.runtimeInfo.CollectorList, " "))
	}

	return nil
}

// Collect implements the interface method
func (collector *OsmCollector) Collect() error {
	clientset, err := kubernetes.NewForConfig(collector.kubeconfig)
	if err != nil {
		return fmt.Errorf("getting access to K8S failed: %w", err)
	}

	// Get all OSM deployments in order to collect information for various resources across all meshes in the cluster
	meshDeploymentList, err := clientset.AppsV1().Deployments("").List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=osm-controller",
	})
	if err != nil {
		return fmt.Errorf("error listing deployments in all namespaces: %w", err)
	}

	for _, deployment := range meshDeploymentList.Items {
		meshName, found := deployment.Labels["meshName"]
		if !found {
			return fmt.Errorf("deployment %s has no 'meshName' label", deployment.Name)
		}

		monitoredNamespaces := []string{}
		monitoredNamespaceList, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("openservicemesh.io/monitored-by=%s", meshName),
		})
		if err != nil {
			// If no monitored namespaces are found, just log and continue - this is not an error.
			if k8sErrors.IsNotFound(err) {
				log.Printf("Failed to find any namespaces monitored by OSM named '%s'\n", meshName)
			}
			return fmt.Errorf("error listing namespaces monitored by OSM named %s: %w", meshName, err)
		} else {
			for _, namespace := range monitoredNamespaceList.Items {
				monitoredNamespaces = append(monitoredNamespaces, namespace.Name)
			}
		}

		collector.callNamespaceCollectors(clientset, monitoredNamespaces, deployment.Namespace, meshName)
		collector.collectGroundTruth(clientset, meshName)
	}

	return nil
}

// callNamespaceCollectors calls functions to collect data for osm-controller namespace and namespaces monitored by a given mesh
func (collector *OsmCollector) callNamespaceCollectors(clientset *kubernetes.Clientset, monitoredNamespaces []string, controllerNamespace string, meshName string) {
	for _, namespace := range monitoredNamespaces {
		if err := collector.collectDataFromEnvoys(clientset, namespace, meshName); err != nil {
			log.Printf("Failed to collect Envoy configs in OSM monitored namespace %s: %+v", namespace, err)
		}
		collector.collectNamespaceResources(namespace, meshName)
	}

	if err := collector.collectPodLogs(clientset, controllerNamespace, meshName); err != nil {
		log.Printf("Failed to collect pod logs for controller namespace %s: %+v", controllerNamespace, err)
	}
	collector.collectNamespaceResources(controllerNamespace, meshName)
}

// collectNamespaceResources collects information about general resources in a given namespace
func (collector *OsmCollector) collectNamespaceResources(namespace string, meshName string) {
	if err := collector.collectPodConfigs(namespace, meshName); err != nil {
		log.Printf("Failed to collect pod configs for ns %s: %+v", namespace, err)
	}

	key := fmt.Sprintf("%s/%s_%s", meshName, namespace, "metadata")
	value, err := collector.commandRunner.GetJsonObjectOutput(&schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}, "", namespace)
	if err != nil {
		value = fmt.Sprintf("Failed to collect metadata for namespace %s: %+v\n", namespace, err)
		log.Print(value)
	}
	collector.data[key] = value

	queryDefinitions := []struct {
		collectorKey string
		schema.GroupVersionResource
		asJson bool
	}{
		{collectorKey: "services_list", GroupVersionResource: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}, asJson: false},
		{collectorKey: "services", GroupVersionResource: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}, asJson: true},
		{collectorKey: "endpoints_list", GroupVersionResource: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"}, asJson: false},
		{collectorKey: "endpoints", GroupVersionResource: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "endpoints"}, asJson: true},
		{collectorKey: "configmaps_list", GroupVersionResource: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}, asJson: false},
		{collectorKey: "configmaps", GroupVersionResource: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}, asJson: true},
		{collectorKey: "ingresses_list", GroupVersionResource: schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}, asJson: false},
		{collectorKey: "ingresses", GroupVersionResource: schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}, asJson: true},
		{collectorKey: "service_accounts_list", GroupVersionResource: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}, asJson: false},
		{collectorKey: "service_accounts", GroupVersionResource: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}, asJson: true},
		{collectorKey: "pods_list", GroupVersionResource: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, asJson: false},
	}

	for _, defn := range queryDefinitions {
		key = fmt.Sprintf("%s/%s_%s", meshName, namespace, defn.collectorKey)
		listOptions := &metav1.ListOptions{}
		if defn.asJson {
			value, err = collector.commandRunner.GetJsonListOutput(&defn.GroupVersionResource, namespace, listOptions)
		} else {
			value, err = collector.commandRunner.GetTableOutput(&defn.GroupVersionResource, namespace, listOptions, &printers.PrintOptions{Wide: true})
		}
		if err != nil {
			value = fmt.Sprintf("Failed to collect %s for namespace %s: %+v\n", defn.GroupVersionResource.Resource, namespace, err)
			log.Print(value)
		}
		collector.data[key] = value
	}
}

// collectPodConfigs collects configs for pods in given namespace
func (collector *OsmCollector) collectPodConfigs(namespace string, meshName string) error {
	listOptions := &metav1.ListOptions{}
	list, err := collector.commandRunner.GetUnstructuredList(&schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, namespace, listOptions)
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		podName := item.GetName()
		value, err := collector.commandRunner.PrintAsJson(&item)
		if err != nil {
			value := fmt.Sprintf("Failed to read JSON for pod %s in %s: %+v\n", podName, namespace, err)
			log.Print(value)
		}
		key := fmt.Sprintf("%s/%s_podConfig", meshName, podName)
		collector.data[key] = value
	}
	return nil
}

// collectDataFromEnvoys collects Envoy proxy config for pods in monitored namespace: port-forward and curl config dump
func (collector *OsmCollector) collectDataFromEnvoys(clientset *kubernetes.Clientset, namespace string, meshName string) error {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		err = collector.portForwardAndRunEnvoyQueries(meshName, namespace, pod.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (collector *OsmCollector) portForwardAndRunEnvoyQueries(meshName, namespace, podName string) error {
	var buffOut, buffErr bytes.Buffer
	readyChan := make(chan struct{})
	stopChan := make(chan struct{}, 1)
	errorChan := make(chan error)
	const localPort = 15000

	defer close(stopChan)

	go func() {
		err := collector.portForward(&portForwardParams{
			namespace: namespace,
			podName:   podName,
			localPort: localPort,
			podPort:   15000,
			outStream: bufio.NewWriter(&buffOut),
			errStream: bufio.NewWriter(&buffErr),
			readyChan: readyChan,
			stopChan:  stopChan,
		})
		if err != nil {
			errorChan <- err
		}
	}()

	select {
	case err := <-errorChan:
		return err
	case <-readyChan:
		collector.runEnvoyQueries(meshName, namespace, podName, localPort)
	}

	return nil
}

func (collector *OsmCollector) runEnvoyQueries(meshName, namespace, podName string, localPort int) {
	envoyQueries := [5]string{"config_dump", "clusters", "listeners", "ready", "stats"}
	for _, query := range envoyQueries {
		queryUrl := fmt.Sprintf("http://localhost:%d/%s", localPort, query)
		responseBody, err := utils.GetUrlWithRetries(queryUrl, 5)
		if err != nil {
			log.Printf("Failed to collect Envoy %s for pod %s in OSM monitored namespace %s: %+v", query, podName, namespace, err)
			continue
		}
		// Remove certificate secrets from Envoy config i.e., "inline_bytes" field from response
		re := regexp.MustCompile("(?m)[\r\n]+^.*inline_bytes.*$")
		secretRemovedResponse := re.ReplaceAllString(string(responseBody), "---redacted---")

		key := fmt.Sprintf("%s/envoy/%s%s", meshName, podName, query)
		collector.data[key] = secretRemovedResponse
	}
}

type portForwardParams struct {
	namespace string
	podName   string
	localPort int
	podPort   int
	outStream io.Writer
	errStream io.Writer
	readyChan chan struct{}
	stopChan  <-chan struct{}
}

func (collector *OsmCollector) portForward(params *portForwardParams) error {
	endpoint, err := url.Parse(collector.kubeconfig.Host)
	if err != nil {
		return fmt.Errorf("error parsing host URL (%s): %w", collector.kubeconfig.Host, err)
	}
	endpoint.Path = fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", params.namespace, params.podName)

	transport, upgrader, err := spdy.RoundTripperFor(collector.kubeconfig)
	if err != nil {
		return err
	}

	portMap := fmt.Sprintf("%d:%d", params.localPort, params.podPort)
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, endpoint)
	fw, err := portforward.New(dialer, []string{portMap}, params.stopChan, params.readyChan, params.outStream, params.errStream)
	if err != nil {
		return err
	}
	return fw.ForwardPorts()
}

// collectPodLogs collects logs of every pod in a given namespace
func (collector *OsmCollector) collectPodLogs(clientset *kubernetes.Clientset, namespace string, meshName string) error {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		output, err := collector.getSinglePodLogs(clientset, namespace, pod.Name)
		if err != nil {
			output = fmt.Sprintf("Failed to collect logs for pod %s: %+v\n", pod.Name, err)
			log.Print(output)
		}
		filePath := meshName + "/" + pod.Name + "_podLogs"
		collector.data[filePath] = output
	}
	return nil
}

func (collector *OsmCollector) getSinglePodLogs(clientset *kubernetes.Clientset, namespace, podName string) (string, error) {
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", fmt.Errorf("error getting log stream for %s/%s", namespace, podName)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error reading log stream for %s/%s", namespace, podName)
	}
	return buf.String(), nil
}

// collectGroundTruth collects ground truth on resources in given mesh
func (collector *OsmCollector) collectGroundTruth(clientset *kubernetes.Clientset, meshName string) {
	type groupVersionResourceKind struct {
		schema.GroupVersionResource
		kind string
	}

	gvrksForGetAll := []groupVersionResourceKind{
		{GroupVersionResource: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}, kind: "Pod"},
		{GroupVersionResource: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}, kind: "Service"},
		{GroupVersionResource: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}, kind: "DaemonSet"},
		{GroupVersionResource: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, kind: "Deployment"},
		{GroupVersionResource: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}, kind: "ReplicaSet"},
		{GroupVersionResource: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}, kind: "StatefulSet"},
		{GroupVersionResource: schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}, kind: "Job"},
		{GroupVersionResource: schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}, kind: "CronJob"},
	}

	gvrksForMutatingWebhookConfig := []groupVersionResourceKind{
		{GroupVersionResource: schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}, kind: "MutatingWebhookConfiguration"},
	}

	gvrksForValidatingWebhookConfig := []groupVersionResourceKind{
		{GroupVersionResource: schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}, kind: "ValidatingWebhookConfiguration"},
	}

	gvrForMeshConfig, err := collector.commandRunner.GetGVRForCRD("meshconfigs.config.openservicemesh.io")
	if err != nil {
		log.Printf("Failed to read MeshConfig CRD: %v", err)
		return
	}

	gvrksForMeshConfig := []groupVersionResourceKind{
		{GroupVersionResource: *gvrForMeshConfig, kind: "MeshConfig"},
	}

	meshLabelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s", meshName)
	queryDefinitions := []struct {
		collectorKey  string
		gvrks         []groupVersionResourceKind
		labelSelector string
		asJson        bool
	}{
		{collectorKey: "all_resources_list", gvrks: gvrksForGetAll, labelSelector: meshLabelSelector, asJson: false},
		{collectorKey: "all_resources_configs", gvrks: gvrksForGetAll, labelSelector: meshLabelSelector, asJson: true},
		{collectorKey: "mutating_webhook_configurations", gvrks: gvrksForMutatingWebhookConfig, labelSelector: meshLabelSelector, asJson: true},
		{collectorKey: "validating_webhook_configurations", gvrks: gvrksForValidatingWebhookConfig, labelSelector: meshLabelSelector, asJson: true},
		{collectorKey: "mesh_configs", gvrks: gvrksForMeshConfig, labelSelector: "", asJson: true},
	}

	for _, defn := range queryDefinitions {
		var sb strings.Builder
		for _, gvrk := range defn.gvrks {
			listOptions := &metav1.ListOptions{LabelSelector: defn.labelSelector}
			var output string
			if defn.asJson {
				output, err = collector.commandRunner.GetJsonListOutput(&gvrk.GroupVersionResource, "", listOptions)
			} else {
				output, err = collector.commandRunner.GetTableOutput(&gvrk.GroupVersionResource, "", listOptions, &printers.PrintOptions{
					Wide:          true,
					WithNamespace: true,
					WithKind:      true,
					Kind:          schema.GroupKind{Group: gvrk.GroupVersionResource.Group, Kind: gvrk.kind},
				})
			}
			if err != nil {
				output = fmt.Sprintf("Error retrieving %s for all namespaces for mesh %s: %+v\n", gvrk.kind, meshName, err)
				log.Print(output)
			}
			sb.WriteString(output)
			sb.WriteString("\n")
		}
		key := fmt.Sprintf("%s/control_plane/%s", meshName, defn.collectorKey)
		collector.data[key] = sb.String()
	}
}

func (collector *OsmCollector) GetData() map[string]string {
	return collector.data
}
