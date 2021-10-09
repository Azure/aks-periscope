package collector

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/Azure/aks-periscope/pkg/utils"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// SmiCollector defines an Smi Collector struct
type SmiCollector struct {
	kubeconfig *restclient.Config
	data       map[string]string
}

// NewSmiCollector is a constructor
func NewSmiCollector(config *restclient.Config) *SmiCollector {
	return &SmiCollector{
		data:       make(map[string]string),
		kubeconfig: config,
	}
}

func (collector *SmiCollector) GetName() string {
	return "smi"
}

func (collector *SmiCollector) GetData() map[string]string {
	return collector.data
}

// Collect implements the interface method
func (collector *SmiCollector) Collect() error {
	// Get all CustomResourceDefinitions in the cluster
	apiextensionsClient, err := apiextensionsclientset.NewForConfig(collector.kubeconfig)
	if err != nil {
		panic(err)
	}

	allCrdsList, err := apiextensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, t := range allCrdsList.Items {
		log.Printf("Test Names foo of %s", t.Name)
	}
	log.Printf("Test list output %s", allCrdsList)

	// Filter to obtain a list of Smi CustomResourceDefinitions in the cluster
	var smiCrdsList []string
	for _, s := range allCrdsList.Items {
		if strings.Contains(s.Name, "smi-spec.io") {
			smiCrdsList = append(smiCrdsList, s.Name)
		}
	}
	if len(smiCrdsList) == 0 {
		return errors.New("cluster does not contain any SMI CRDs")
	}

	collectSmiCrds(collector, smiCrdsList, apiextensionsClient)
	return collectSmiCustomResourcesFromAllNamespaces(collector, smiCrdsList)
}

func collectSmiCrds(collector *SmiCollector, smiCrdsList []string, apiextensionsClient *apiextensionsclientset.Clientset) {

	for _, smiCrd := range smiCrdsList {
		yamlDefinition, err := apiextensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(context.Background(), smiCrd, metav1.GetOptions{})

		if err != nil {
			log.Printf("Skipping: unable to collect yaml definition of %s: %+v", smiCrd, err)
		}

		log.Printf("Test -----<> %v ", yamlDefinition)
		collector.data["smi/crd_"+strings.TrimSuffix(smiCrd, ".io")] = yamlDefinition.String()

	}
}

func collectSmiCustomResourcesFromAllNamespaces(collector *SmiCollector, smiCrdsList []string) error {
	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(collector.kubeconfig)
	if err != nil {
		return fmt.Errorf("getting access to K8S failed: %w", err)
	}
	namespacesList, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("collect SMI custom resources: unable to list namespaces in the cluster: %w", err)
	}

	// Get all namespaces in the cluster
	for _, namespace := range namespacesList.Items {
		collectSmiCustomResourcesFromNamespace(collector, smiCrdsList, namespace.Namespace)
	}

	return nil
}

func collectSmiCustomResourcesFromNamespace(collector *SmiCollector, smiCrdsList []string, namespace string) {
	for _, smiCrdType := range smiCrdsList {
		// TODO: Note - this part needs to be carefully tackle by folks who know exact working of this and expectations.

		// get all custom resources of this smi crd type
		customResourcesList, err := utils.GetResourceList([]string{"get", smiCrdType, "-n", namespace, "-o", "jsonpath={..metadata.name}"}, " ")
		if err != nil {
			log.Printf("Skipping: unable to list custom resources of type %s in namespace %s: %+v", smiCrdType, namespace, err)
			continue
		}

		for _, customResourceName := range customResourcesList {
			yamlDefinition, err := utils.RunCommandOnContainer("kubectl", "get", smiCrdType, customResourceName, "-n", namespace, "-o", "yaml")
			if err != nil {
				log.Printf("Skipping: unable to collect yaml definition of %s (custom resource type: %s): %+v", customResourceName, smiCrdType, err)
			}
			collector.data["smi/namespace_"+namespace+"/"+smiCrdType+"_"+customResourceName+"_custom_resource"] = yamlDefinition
		}
	}
}
