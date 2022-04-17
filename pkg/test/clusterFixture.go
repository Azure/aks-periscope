package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
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
	NamespaceSuffix string
	CommandRunner   *ToolsCommandRunner
	ClientConfig    *rest.Config
	Clientset       *kubernetes.Clientset
	KubeConfigFile  *os.File
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
	if fixture.KubeConfigFile != nil {
		os.Remove(fixture.KubeConfigFile.Name())
	}
}

func buildInstance() (*ClusterFixture, error) {
	fixture := &ClusterFixture{
		NamespaceSuffix: time.Now().UTC().Format("20060102-150405"),
	}

	client, err := client.NewEnvClient()
	if err != nil {
		return fixture, fmt.Errorf("Unable to create docker client: %v", err)
	}

	toolsImageBuilder := NewToolsImageBuilder(client)
	err = toolsImageBuilder.Build()
	if err != nil {
		return fixture, fmt.Errorf("Error building tools image: %v", err)
	}

	fixture.CommandRunner = NewToolsCommandRunner(client)

	createClusterCommand := GetCreateClusterCommand()
	kubeConfigContent, err := fixture.CommandRunner.Run(createClusterCommand)
	if err != nil {
		return fixture, fmt.Errorf("Error creating cluster: %v", err)
	}

	kubeConfigContentBytes := []byte(kubeConfigContent)
	config, err := clientcmd.NewClientConfigFromBytes(kubeConfigContentBytes)
	if err != nil {
		return fixture, fmt.Errorf("Error reading kubeconfig: %v", err)
	}

	fixture.ClientConfig, err = config.ClientConfig()
	if err != nil {
		return fixture, fmt.Errorf("Error creating client config from config: %v", err)
	}

	fixture.Clientset, err = kubernetes.NewForConfig(fixture.ClientConfig)
	if err != nil {
		return fixture, fmt.Errorf("Failed to create client connection to kubernetes from kubeconfig: %v", err)
	}

	err = cleanTestNamespaces(fixture.Clientset)
	if err != nil {
		return fixture, fmt.Errorf("Error cleaning test namespaces: %v", err)
	}

	fixture.KubeConfigFile, err = ioutil.TempFile("", "")
	_, err = fixture.KubeConfigFile.Write(kubeConfigContentBytes)
	if err != nil {
		return fixture, fmt.Errorf("Error creating kubeconfig file %s: %v", fixture.KubeConfigFile.Name(), err)
	}
	err = fixture.KubeConfigFile.Close()
	if err != nil {
		return fixture, fmt.Errorf("Error closing kubeconfig file %s: %v", fixture.KubeConfigFile.Name(), err)
	}

	return fixture, nil
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

func (fixture *ClusterFixture) CreateNamespace(prefix string) (string, error) {
	name := fmt.Sprintf("%s-%s", prefix, fixture.NamespaceSuffix)
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": testingLabel,
			},
		},
	}

	_, err := fixture.Clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("Error creating namespace %s: %v", name, err)
	}
	return name, nil
}
