package collector

import (
	"fmt"
	"log"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/kubectl/pkg/describe"
)

// KubeObjectsCollector defines a KubeObjects Collector struct
type KubeObjectsCollector struct {
	data          map[string]string
	kubeconfig    *restclient.Config
	commandRunner *utils.KubeCommandRunner
	runtimeInfo   *utils.RuntimeInfo
}

// NewKubeObjectsCollector is a constructor
func NewKubeObjectsCollector(config *restclient.Config, runtimeInfo *utils.RuntimeInfo) *KubeObjectsCollector {
	return &KubeObjectsCollector{
		data:          make(map[string]string),
		kubeconfig:    config,
		commandRunner: utils.NewKubeCommandRunner(config),
		runtimeInfo:   runtimeInfo,
	}
}

func (collector *KubeObjectsCollector) GetName() string {
	return "kubeobjects"
}

func (collector *KubeObjectsCollector) CheckSupported() error {
	return nil
}

// Collect implements the interface method
func (collector *KubeObjectsCollector) Collect() error {
	// Create a discovery client for querying resource metadata
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(collector.kubeconfig)
	if err != nil {
		return fmt.Errorf("error creating discovery client: %w", err)
	}

	// Create a RESTMapper to handle the mapping between GroupKind and GroupVersionResource
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))

	for _, kubernetesObject := range collector.runtimeInfo.KubernetesObjects {
		kubernetesObjectParts := strings.Split(kubernetesObject, "/")
		if len(kubernetesObjectParts) < 2 {
			log.Printf("Invalid kube-objects value: %s", kubernetesObject)
			continue
		}

		namespace := kubernetesObjectParts[0]
		groupResource := schema.ParseGroupResource(kubernetesObjectParts[1])

		groupVersionKind, err := mapper.KindFor(groupResource.WithVersion(""))
		if err != nil {
			log.Printf("Unable to determine Kind for resource %s: %v", groupResource.String(), err)
			continue
		}

		describer, ok := describe.DescriberFor(groupVersionKind.GroupKind(), collector.kubeconfig)
		if !ok {
			log.Printf("Unable to create Describer for Kind %s", groupVersionKind.String())
			continue
		}

		// Get the resources within the namespace to describe
		var resourceNames []string
		if len(kubernetesObjectParts) > 2 {
			resourceNames = []string{kubernetesObjectParts[2]}
		} else {
			resourceNames, err = collector.getResourcesInNamespace(mapper, &groupResource, namespace)
			if err != nil {
				log.Printf("Unable to get %s resources in %s: %v", groupResource.String(), namespace, err)
				continue
			}
		}

		for _, resourceName := range resourceNames {
			output, err := describer.Describe(namespace, resourceName, describe.DescriberSettings{ShowEvents: true})
			if err != nil {
				log.Printf("Error describing %s %s in namespace %s: %v", groupVersionKind.String(), resourceName, namespace, err)
				continue
			}

			key := fmt.Sprintf("%s_%s_%s", namespace, groupResource.String(), resourceName)
			collector.data[key] = output
		}
	}

	return nil
}

func (collector *KubeObjectsCollector) getResourcesInNamespace(mapper meta.RESTMapper, groupResource *schema.GroupResource, namespace string) ([]string, error) {
	groupVersionResource, err := mapper.ResourceFor(groupResource.WithVersion(""))
	if err != nil {
		return []string{}, fmt.Errorf("error determining Version for resource %s: %v", groupResource.String(), err)
	}

	resources, err := collector.commandRunner.GetUnstructuredList(&groupVersionResource, namespace, &metav1.ListOptions{})
	if err != nil {
		return []string{}, fmt.Errorf("error listing %s: %v", groupVersionResource.String(), err)
	}

	resourceNames := make([]string, len(resources.Items))
	for i, resource := range resources.Items {
		resourceNames[i] = resource.GetName()
	}

	return resourceNames, nil
}

func (collector *KubeObjectsCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}
