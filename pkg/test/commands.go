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

func GetInstallMetricsServerCommand(hostKubeconfigPath string) (string, []string) {
	kubeConfigPath := "/.kube/config"
	installCommand := fmt.Sprintf("kubectl --kubeconfig=%s apply -f /resources/metrics-server/components.yaml", kubeConfigPath)
	waitDeployCommand := fmt.Sprintf("kubectl wait --kubeconfig=%s --for condition=Available=True deployment -n kube-system metrics-server --timeout=240s", kubeConfigPath)
	waitPodsCommand := fmt.Sprintf("kubectl --kubeconfig=%s wait --for condition=ready pod -n kube-system -l k8s-app=metrics-server --timeout=240s", kubeConfigPath)
	command := fmt.Sprintf("%s && %s && %s", installCommand, waitDeployCommand, waitPodsCommand)
	return command, []string{
		fmt.Sprintf("%s:%s", hostKubeconfigPath, kubeConfigPath),
	}
}

func GetInstallHelmChartCommand(name, namespace, hostKubeconfigPath string) (string, []string) {
	kubeConfigPath := "/.kube/config"
	command := fmt.Sprintf("KUBECONFIG=%s helm install %s /resources/testchart --namespace %s --create-namespace", kubeConfigPath, name, namespace)
	return command, []string{
		fmt.Sprintf("%s:%s", hostKubeconfigPath, kubeConfigPath),
	}
}
