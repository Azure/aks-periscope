package collector

import (
	"fmt"
	"strings"

	"github.com/Azure/aks-periscope/pkg/utils"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/describe"
)

// KubeObjectsCollector defines a KubeObjects Collector struct
type KubeObjectsCollector struct {
	data        map[string]string
	kubeconfig  *restclient.Config
	runtimeInfo *utils.RuntimeInfo
}

// NewKubeObjectsCollector is a constructor
func NewKubeObjectsCollector(config *restclient.Config, runtimeInfo *utils.RuntimeInfo) *KubeObjectsCollector {
	return &KubeObjectsCollector{
		data:        make(map[string]string),
		kubeconfig:  config,
		runtimeInfo: runtimeInfo,
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
	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(collector.kubeconfig)
	if err != nil {
		return fmt.Errorf("getting access to K8S failed: %w", err)
	}

	for _, kubernetesObject := range collector.runtimeInfo.KubernetesObjects {
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
