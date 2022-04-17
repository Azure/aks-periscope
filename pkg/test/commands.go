package test

import "fmt"

const (
	TestClusterName = "aks-periscope-testing"
	KindNodeTag     = "v1.23.5" // https://hub.docker.com/r/kindest/node/tags
)

func GetCreateClusterCommand() string {
	existsClusterCommand := fmt.Sprintf("kind get clusters | grep -q '^%s$'", TestClusterName)
	createClusterCommand := fmt.Sprintf("kind create cluster --name %s --image kindest/node:%s", TestClusterName, KindNodeTag)
	getKubeConfigCommand := fmt.Sprintf("kind get kubeconfig --name %s", TestClusterName)
	return fmt.Sprintf("%s || %s && %s", existsClusterCommand, createClusterCommand, getKubeConfigCommand)
}

func GetInstallHelmChartCommand(name, namespace, hostChartPath, hostKubeconfigPath string) (string, []string) {
	chartPath := "/testchart"
	kubeConfigPath := "/.kube/config"
	command := fmt.Sprintf("KUBECONFIG=%s helm install %s %s --namespace %s --create-namespace", kubeConfigPath, name, chartPath, namespace)
	return command, []string{
		fmt.Sprintf("%s:%s", hostChartPath, chartPath),
		fmt.Sprintf("%s:%s", hostKubeconfigPath, kubeConfigPath),
	}
}
