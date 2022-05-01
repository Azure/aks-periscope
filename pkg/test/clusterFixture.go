package test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const testingLabel = "aks-periscope-test"

var once sync.Once

// ClusterFixture holds all information required to connect to a local cluster, generated on the fly
// for testing purposes. It supports running arbitrary command-line tools available via a locally-built
// Docker image containing any desired tools for test setup.
type ClusterFixture struct {
	NamespaceSuffix string
	CommandRunner   *ToolsCommandRunner
	ClientConfig    *rest.Config
	Clientset       *kubernetes.Clientset
	KubeConfigFile  *os.File
}

var fixtureInstance *ClusterFixture
var fixtureError error

// GetClusterFixture can be called from test files, and will always return the same instance of the Fixture
// (per test process).
func GetClusterFixture() (*ClusterFixture, error) {
	if fixtureInstance == nil {
		once.Do(
			func() {
				fixtureInstance, fixtureError = buildInstance()
			})
	}

	return fixtureInstance, fixtureError
}

// Cleanup is intended to be called after all tests have run. It does not delete the cluster itself, because
// re-creating it is an expensive operation, and the goal here is to allow fast re-runs when testing locally.
func (fixture *ClusterFixture) Cleanup() {
	// Assume errors will not be handled by caller - just log them here and continue
	if fixture.Clientset != nil && fixture.CommandRunner != nil && fixture.KubeConfigFile != nil {
		err := cleanupResources(fixture.Clientset, fixture.CommandRunner, fixture.KubeConfigFile)
		if err != nil {
			log.Printf("Error cleaning up resources: %v", err)
		}
	}

	if fixture.KubeConfigFile != nil {
		kubeConfigFileName := fixture.KubeConfigFile.Name()
		err := os.Remove(kubeConfigFileName)
		if err != nil {
			log.Printf("Error deleting kubeconfig file %s: %v", kubeConfigFileName, err)
		}
	}
}

func buildInstance() (*ClusterFixture, error) {
	fixture := &ClusterFixture{
		NamespaceSuffix: time.Now().UTC().Format("20060102-150405"),
	}

	client, err := client.NewClientWithOpts()
	if err != nil {
		return fixture, fmt.Errorf("unable to create docker client: %w", err)
	}

	toolsImageBuilder := NewToolsImageBuilder(client)
	err = toolsImageBuilder.Build()
	if err != nil {
		return fixture, fmt.Errorf("error building tools image: %w", err)
	}

	fixture.CommandRunner = NewToolsCommandRunner(client)

	createClusterCommand := GetCreateClusterCommand()
	kubeConfigContent, err := fixture.CommandRunner.Run(createClusterCommand)
	if err != nil {
		return fixture, fmt.Errorf("error creating cluster: %w", err)
	}

	err = PullAndLoadDockerImages(client, fixture.CommandRunner)
	if err != nil {
		return fixture, fmt.Errorf("error pulling and loading Docker images: %w", err)
	}

	kubeConfigContentBytes := []byte(kubeConfigContent)
	config, err := clientcmd.NewClientConfigFromBytes(kubeConfigContentBytes)
	if err != nil {
		return fixture, fmt.Errorf("error reading kubeconfig: %w", err)
	}

	fixture.ClientConfig, err = config.ClientConfig()
	if err != nil {
		return fixture, fmt.Errorf("error creating client config from config: %w", err)
	}

	fixture.Clientset, err = kubernetes.NewForConfig(fixture.ClientConfig)
	if err != nil {
		return fixture, fmt.Errorf("failed to create client connection to kubernetes from kubeconfig: %w", err)
	}

	fixture.KubeConfigFile, err = ioutil.TempFile("", "")
	if err != nil {
		return fixture, fmt.Errorf("error creating temp file for kubeconfig: %w", err)
	}
	_, err = fixture.KubeConfigFile.Write(kubeConfigContentBytes)
	if err != nil {
		return fixture, fmt.Errorf("error creating kubeconfig file %s: %w", fixture.KubeConfigFile.Name(), err)
	}
	err = fixture.KubeConfigFile.Close()
	if err != nil {
		return fixture, fmt.Errorf("error closing kubeconfig file %s: %w", fixture.KubeConfigFile.Name(), err)
	}

	// Now we have a kubeconfig and cluster, cleanup any leftovers within the cluster from previous tests
	err = cleanupResources(fixture.Clientset, fixture.CommandRunner, fixture.KubeConfigFile)
	if err != nil {
		return fixture, fmt.Errorf("error cleaning up resources: %w", err)
	}

	// Install shared cluster resources
	err = installResources(fixture.Clientset, fixture.CommandRunner, fixture.KubeConfigFile)
	if err != nil {
		return fixture, fmt.Errorf("error installing resources: %w", err)
	}

	return fixture, nil
}

// PrintDiagnostics logs information to stdout that might be helpful for diagnosing test failures
// (particularly helpful in a CI environment where it is not possible to break execution with a debugger).
func (fixture *ClusterFixture) PrintDiagnostics() {
	diagnosticsCommand, binds := GetTestDiagnosticsCommand(fixture.KubeConfigFile.Name())
	diagnosticsOutput, err := fixture.CommandRunner.Run(diagnosticsCommand, binds...)
	fmt.Println(diagnosticsOutput)
	if err != nil {
		fmt.Printf("error running test diagnostics command: %v", err)
	}
}

func installResources(clientset *kubernetes.Clientset, commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	err := InstallMetricsServer(commandRunner, kubeConfigFile)
	if err != nil {
		return fmt.Errorf("error installing metrics server: %w", err)
	}

	return nil
}

func cleanupResources(clientset *kubernetes.Clientset, commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	// We only bother to clean up those resources which would cause problems next time we try and install
	err := CleanTestNamespaces(clientset)
	if err != nil {
		return err
	}
	return nil
}

func (fixture *ClusterFixture) CreateNamespace(name string) error {
	return CreateTestNamespace(fixture.Clientset, name)
}
