package collector

import (
	"fmt"
	"os"
	"strings"

	"github.com/Azure/aks-periscope/pkg/utils"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/describe"
)

// KubeObjectsCollector defines a KubeObjects Collector struct
type KubeObjectsCollector struct {
	kubeconfig *restclient.Config
	data       map[string]string
}

// NewKubeObjectsCollector is a constructor
func NewKubeObjectsCollector(config *restclient.Config) *KubeObjectsCollector {
	return &KubeObjectsCollector{
		data:       make(map[string]string),
		kubeconfig: config,
	}
}

func (collector *KubeObjectsCollector) GetName() string {
	return "kubeobjects"
}

// Collect implements the interface method
func (collector *KubeObjectsCollector) Collect() error {
	kubernetesObjects := strings.Fields(os.Getenv("DIAGNOSTIC_KUBEOBJECTS_LIST"))

	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(collector.kubeconfig)
	if err != nil {
		return fmt.Errorf("getting access to K8S failed: %w", err)
	}

	for _, kubernetesObject := range kubernetesObjects {
		kubernetesObjectParts := strings.Split(kubernetesObject, "/")
		nameSpace := kubernetesObjectParts[0]
		objectType := kubernetesObjectParts[1]

		// List the pods in the given namespace
		podList, err := utils.GetPods(clientset, nameSpace)
		if err != nil {
			return fmt.Errorf("getting pods failed: %w", err)
		}

		for _, pod := range podList.Items {
			d := describe.PodDescriber{
				Interface: clientset,
			}

			output, err := d.Describe(pod.Namespace, pod.Name, describe.DescriberSettings{
				ShowEvents: true,
			})
			if err != nil {
				return fmt.Errorf("getting description failed: %w", err)
			}

			collector.data[pod.Namespace+"_"+objectType+"_"+pod.Name] = output
		}
	}

	return nil
}

func (collector *KubeObjectsCollector) GetData() map[string]string {
	return collector.data
}
