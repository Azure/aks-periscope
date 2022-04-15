package test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/client"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const testingLabel = "aks-periscope-test"

var once sync.Once

type ClusterFixture struct {
	Namespace    string
	ClientConfig *rest.Config
	Clientset    *kubernetes.Clientset
}

var fixtureInstance *ClusterFixture
var fixtureError error

func GetClusterFixture() (*ClusterFixture, error) {
	if fixtureInstance == nil {
		once.Do(
			func() {
				fixtureInstance, fixtureError = buildInstance()
				if fixtureError != nil {
					fixtureInstance = &ClusterFixture{}
				}
			})
	}

	if fixtureError != nil {
		return nil, fixtureError
	}

	return fixtureInstance, nil
}

func (fixture *ClusterFixture) Cleanup() {
	cleanTestNamespaces(fixture.Clientset)
}

func buildInstance() (*ClusterFixture, error) {
	client, err := client.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("Unable to create docker client: %v", err)
	}

	toolsImageBuilder := NewToolsImageBuilder(client)
	err = toolsImageBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("Error building tools image: %v", err)
	}

	commandRunner := NewToolsCommandRunner(client)

	createClusterCommand := GetCreateClusterCommand()
	kubeConfigContent, err := commandRunner.Run(createClusterCommand)
	if err != nil {
		return nil, fmt.Errorf("Error creating cluster: %v", err)
	}

	config, err := clientcmd.NewClientConfigFromBytes([]byte(kubeConfigContent))
	if err != nil {
		return nil, fmt.Errorf("Error reading kubeconfig: %v", err)
	}

	clientConfig, err := config.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("Error creating client config from config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to create client connection to kubernetes from kubeconfig: %v", err)
	}

	timeString := time.Now().UTC().Format("20060102-150405")
	namespace := fmt.Sprintf("test-%s", timeString)

	err = cleanTestNamespaces(clientset)
	if err != nil {
		return nil, fmt.Errorf("Error cleaning test namespaces: %v", err)
	}
	err = createNamespace(clientset, namespace)
	if err != nil {
		return nil, fmt.Errorf("Error creating namespace %s: %v", namespace, err)
	}

	return &ClusterFixture{
		Namespace:    namespace,
		ClientConfig: clientConfig,
		Clientset:    clientset,
	}, nil
}

func cleanTestNamespaces(clientset *kubernetes.Clientset) error {
	namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", testingLabel),
	})
	if err != nil {
		return fmt.Errorf("Error listing namespaces: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(namespaceList.Items))
	for _, namespace := range namespaceList.Items {
		go func(name string) {
			defer wg.Done()
			err = clientset.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
		}(namespace.Name)
	}

	wg.Wait()
	return err
}

func createNamespace(clientset *kubernetes.Clientset, name string) error {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": testingLabel,
			},
		},
	}

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("Error creating namespace %s: %v", name, err)
	}
	return nil
}
