package test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	testClusterName = "aks-periscope-testing"
	kindNodeTag     = "v1.23.5" // https://hub.docker.com/r/kindest/node/tags
	kubeConfigPath  = "/root/.kube/config"
)

func GetCreateClusterCommand() string {
	// Create the cluster if it doesn't exist, and output the kubeconfig content
	existsClusterCommand := fmt.Sprintf("kind get clusters | grep -q '^%s$'", testClusterName)
	createClusterCommand := fmt.Sprintf("kind create cluster --name %s --config=/resources/kind-config/config.yaml --wait 5m --image kindest/node:%s", testClusterName, kindNodeTag)
	getKubeConfigCommand := fmt.Sprintf("kind get kubeconfig --name %s", testClusterName)
	return fmt.Sprintf("%s || %s && %s", existsClusterCommand, createClusterCommand, getKubeConfigCommand)
}

func GetLoadDockerImagesCommand(images []string) string {
	return fmt.Sprintf(`echo "%s" | xargs -P8 -n1 kind load docker-image --name %s`, strings.Join(images, " "), testClusterName)
}

func GetInstallMetricsServerCommand(hostKubeconfigPath string) (string, []string) {
	installCommand := "kubectl apply -f /resources/metrics-server/components.yaml"
	waitCommand := "kubectl rollout status -n kube-system deploy/metrics-server --timeout=240s"
	command := fmt.Sprintf("%s && %s", installCommand, waitCommand)
	return command, []string{getBinding(hostKubeconfigPath, kubeConfigPath)}
}

func GetInstallHelmChartCommand(name, namespace, hostKubeconfigPath string) (string, []string) {
	command := fmt.Sprintf("helm install %s /resources/testchart --namespace %s", name, namespace)
	return command, []string{getBinding(hostKubeconfigPath, kubeConfigPath)}
}

func GetTestDiagnosticsCommand(hostKubeconfigPath string) (string, []string) {
	commands := []string{
		"printf '\nDESCRIBE NODES\n'",
		"kubectl describe node",
		"printf '\nTOP NODES\n'",
		"kubectl top node",
		"printf '\nTOP PODS\n'",
		"kubectl top pod -A",
		"printf '\nALL PODS\n'",
		"kubectl get pod -A",
	}

	return strings.Join(commands, " && "), []string{getBinding(hostKubeconfigPath, kubeConfigPath)}
}

func getBinding(hostPath, containerPath string) string {
	if runtime.GOOS == "windows" {
		hostPath = strings.Replace(filepath.ToSlash(hostPath), "C:", "/c", 1)
	}

	return fmt.Sprintf("%s:%s", hostPath, containerPath)
}
