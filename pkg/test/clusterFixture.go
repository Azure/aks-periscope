package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	testClusterName   = "aks-periscope-testing"
	kindNodeTag       = "v1.23.5" // https://hub.docker.com/r/kindest/node/tags
	kubeConfigPath    = "/root/.kube/config"
	osmVersion        = "1.1.0"
	testingLabelValue = "aks-periscope-test"
	meshName          = "test-osm" // used for both the helm release name, *and* the mesh name referred to by the CLI (e.g. for adding namespaces)
)

var once sync.Once

// ClusterAccess groups the objects used for connecting to a cluster as a single user/serviceaccount.
type ClusterAccess struct {
	ClientConfig   *rest.Config
	Clientset      *kubernetes.Clientset
	KubeConfigFile *os.File
}

// ClusterFixture holds all information required to connect to a local cluster, generated on the fly
// for testing purposes. It supports running arbitrary command-line tools available via a locally-built
// Docker image containing any desired tools for test setup.
type ClusterFixture struct {
	NamespaceSuffix string
	KnownNamespaces *KnownNamespaces
	CommandRunner   *ToolsCommandRunner
	AdminAccess     *ClusterAccess
	PeriscopeAccess *ClusterAccess
}

type KnownNamespaces struct {
	OsmSystem        string
	OsmBookBuyer     string
	OsmBookStore     string
	OsmBookThief     string
	OsmBookWarehouse string
	Periscope        string
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

// CreateTestNamespace creates a Kuberenetes namespace with a suffix that changes for each test run,and a well-known label.
// The label is used for cleanup purposes, so that it is easy to identify which namespaces have been created for testing and delete
// just those. The suffix ensures that different namespace resources will be created on each test run, meaning a test run won't
// be impacted by slow deletion of namespaces from previous runs.
func (fixture *ClusterFixture) CreateTestNamespace(prefix string) (string, error) {
	namespace := getTestNamespace(prefix, fixture.NamespaceSuffix)
	err := createTestNamespace(fixture.AdminAccess.Clientset, namespace)
	return namespace, err
}

// CheckDockerImages checks our list of required images is up-to-date based on images stored in the cluster's nodes.
// If any images are superfluous or missing it will return an error specifying the image tags that need to be added or removed.
// It also verifies the pull policies to ensure that no unnecessary downloading of images occurs during test runs.
func (fixture *ClusterFixture) CheckDockerImages() error {
	return checkDockerImages(fixture.AdminAccess.Clientset)
}

// PrintDiagnostics logs information to stdout that might be helpful for diagnosing test failures
// (particularly helpful in a CI environment where it is not possible to break execution with a debugger).
func (fixture *ClusterFixture) PrintDiagnostics() {
	diagnosticsCommand, binds := getTestDiagnosticsCommand(fixture.AdminAccess.KubeConfigFile.Name())
	diagnosticsOutput, err := fixture.CommandRunner.Run(diagnosticsCommand, binds...)
	fmt.Println(diagnosticsOutput)
	if err != nil {
		fmt.Printf("error running test diagnostics command: %v", err)
	}
}

// GetNodeNames retrieves the names of the nodes in the test cluster.
func (fixture *ClusterFixture) GetNodeNames() ([]string, error) {
	nodeList, err := fixture.AdminAccess.Clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing nodes: %w", err)
	}

	nodeNames := make([]string, len(nodeList.Items))
	for i, node := range nodeList.Items {
		nodeNames[i] = node.Name
	}

	return nodeNames, nil
}

// GetKubeConfigBinding gets the Docker volume binding required to map the fixture's kubeconfig file
// to the expected location in the testing tools container.
func (clusterAccess *ClusterAccess) GetKubeConfigBinding() string {
	return getKubeConfigBinding(clusterAccess.KubeConfigFile.Name())
}

// Cleanup is intended to be called after all tests have run. It does not delete the cluster itself, because
// re-creating it is an expensive operation, and the goal here is to allow fast re-runs when testing locally.
func (fixture *ClusterFixture) Cleanup() {
	// Assume errors will not be handled by caller - just log them here and continue
	if fixture.PeriscopeAccess != nil {
		cleanupFile(fixture.PeriscopeAccess.KubeConfigFile)
	}

	if fixture.AdminAccess != nil {
		if fixture.AdminAccess.Clientset != nil && fixture.CommandRunner != nil && fixture.AdminAccess.KubeConfigFile != nil {
			err := cleanupResources(fixture.AdminAccess.Clientset, fixture.CommandRunner, fixture.AdminAccess.KubeConfigFile)
			if err != nil {
				log.Printf("Error cleaning up resources: %v", err)
			}
		}

		cleanupFile(fixture.AdminAccess.KubeConfigFile)
	}
}

func cleanupFile(file *os.File) {
	// Assume errors will not be handled by caller - just log them here and continue
	if file != nil {
		fileName := file.Name()
		err := os.Remove(fileName)
		if err != nil {
			log.Printf("Error deleting file %s: %v", fileName, err)
		}
	}
}

func buildInstance() (*ClusterFixture, error) {
	namespaceSuffix := time.Now().UTC().Format("20060102-150405")
	fixture := &ClusterFixture{
		NamespaceSuffix: namespaceSuffix,
		KnownNamespaces: &KnownNamespaces{
			OsmSystem:        getTestNamespace("osm", namespaceSuffix),
			OsmBookBuyer:     getTestNamespace("bookbuyer", namespaceSuffix),
			OsmBookStore:     getTestNamespace("bookstore", namespaceSuffix),
			OsmBookThief:     getTestNamespace("bookthief", namespaceSuffix),
			OsmBookWarehouse: getTestNamespace("bookwarehouse", namespaceSuffix),
			Periscope:        getTestNamespace("aks-periscope", namespaceSuffix),
		},
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

	createClusterCommand := getCreateClusterCommand()
	adminKubeConfigContent, err := fixture.CommandRunner.Run(createClusterCommand)
	if err != nil {
		return fixture, fmt.Errorf("error creating cluster: %w", err)
	}

	err = pullAndLoadDockerImages(client, fixture.CommandRunner)
	if err != nil {
		return fixture, fmt.Errorf("error pulling and loading Docker images: %w", err)
	}

	fixture.AdminAccess, err = createClusterAccess([]byte(adminKubeConfigContent))
	if err != nil {
		return fixture, fmt.Errorf("error creating admin access to cluster: %w", err)
	}

	// Now we have a kubeconfig and cluster, cleanup any leftovers within the cluster from previous tests
	err = cleanupResources(fixture.AdminAccess.Clientset, fixture.CommandRunner, fixture.AdminAccess.KubeConfigFile)
	if err != nil {
		return fixture, fmt.Errorf("error cleaning up resources: %w", err)
	}

	// Create Periscope deployment resources
	err = createTestNamespace(fixture.AdminAccess.Clientset, fixture.KnownNamespaces.Periscope)
	if err != nil {
		return fixture, fmt.Errorf("error creating Periscope namespace %s: %w", fixture.KnownNamespaces.Periscope, err)
	}

	err = deployPeriscopeServiceAccount(fixture.CommandRunner, fixture.AdminAccess.KubeConfigFile, fixture.KnownNamespaces.Periscope)
	if err != nil {
		return fixture, fmt.Errorf("error deploying Periscope service account: %w", err)
	}

	periscopeServiceAccountKubeconfigCommand, binds := getPeriscopeServiceAccountKubeconfigCommand(fixture.AdminAccess.KubeConfigFile.Name(), fixture.KnownNamespaces.Periscope)
	periscopeKubeConfigContent, err := fixture.CommandRunner.Run(periscopeServiceAccountKubeconfigCommand, binds...)
	if err != nil {
		return fixture, fmt.Errorf("error getting kubeconfig for Periscope SA user: %w", err)
	}

	fixture.PeriscopeAccess, err = createClusterAccess([]byte(periscopeKubeConfigContent))
	if err != nil {
		return fixture, fmt.Errorf("error creating Periscope access to cluster: %w", err)
	}

	// Install shared cluster resources
	err = installResources(fixture.AdminAccess.Clientset, fixture.CommandRunner, fixture.AdminAccess.KubeConfigFile, fixture.KnownNamespaces)
	if err != nil {
		return fixture, fmt.Errorf("error installing resources: %w", err)
	}

	return fixture, nil
}

func createClusterAccess(kubeConfigContentBytes []byte) (*ClusterAccess, error) {
	clusterAccess := &ClusterAccess{}

	config, err := clientcmd.NewClientConfigFromBytes(kubeConfigContentBytes)
	if err != nil {
		return clusterAccess, fmt.Errorf("error reading kubeconfig: %w", err)
	}

	clusterAccess.ClientConfig, err = config.ClientConfig()
	if err != nil {
		return clusterAccess, fmt.Errorf("error creating client config from config: %w", err)
	}

	clusterAccess.Clientset, err = kubernetes.NewForConfig(clusterAccess.ClientConfig)
	if err != nil {
		return clusterAccess, fmt.Errorf("failed to create client connection to kubernetes from kubeconfig: %w", err)
	}

	clusterAccess.KubeConfigFile, err = os.CreateTemp("", "")
	if err != nil {
		return clusterAccess, fmt.Errorf("error creating temp file for kubeconfig: %w", err)
	}

	_, err = clusterAccess.KubeConfigFile.Write(kubeConfigContentBytes)
	if err != nil {
		return clusterAccess, fmt.Errorf("error creating kubeconfig file %s: %w", clusterAccess.KubeConfigFile.Name(), err)
	}

	err = clusterAccess.KubeConfigFile.Close()
	if err != nil {
		return clusterAccess, fmt.Errorf("error closing kubeconfig file %s: %w", clusterAccess.KubeConfigFile.Name(), err)
	}

	return clusterAccess, nil
}

func installResources(clientset *kubernetes.Clientset, commandRunner *ToolsCommandRunner, kubeConfigFile *os.File, knownNamespaces *KnownNamespaces) error {
	err := installMetricsServer(commandRunner, kubeConfigFile)
	if err != nil {
		return fmt.Errorf("error installing metrics server: %w", err)
	}

	err = installOsm(clientset, commandRunner, kubeConfigFile, knownNamespaces.OsmSystem)
	if err != nil {
		return fmt.Errorf("error installing OSM: %w", err)
	}

	err = deployOsmApplications(clientset, commandRunner, kubeConfigFile, knownNamespaces)
	if err != nil {
		return fmt.Errorf("error deploying OSM applications: %w", err)
	}

	return nil
}

func cleanupResources(clientset *kubernetes.Clientset, commandRunner *ToolsCommandRunner, kubeConfigFile *os.File) error {
	// We only bother to clean up those resources which would cause problems next time we try and install
	err := uninstallHelmReleases(commandRunner, kubeConfigFile)
	if err != nil {
		return err
	}
	err = cleanTestNamespaces(clientset)
	if err != nil {
		return err
	}
	return nil
}

func getTestNamespace(prefix, suffix string) string { return fmt.Sprintf("%s-%s", prefix, suffix) }
