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
	osmName         = "test-osm"
)

func GetCreateClusterCommand() string {
	existsClusterCommand := fmt.Sprintf("kind get clusters | grep -q '^%s$'", testClusterName)
	createClusterCommand := fmt.Sprintf("kind create cluster --name %s --image kindest/node:%s", testClusterName, kindNodeTag)
	getKubeConfigCommand := fmt.Sprintf("kind get kubeconfig --name %s", testClusterName)
	return fmt.Sprintf("%s || %s && %s", existsClusterCommand, createClusterCommand, getKubeConfigCommand)
}

func GetInstallMetricsServerCommand(hostKubeconfigPath string) (string, []string) {
	installCommand := "kubectl apply -f /resources/metrics-server/components.yaml"
	waitDeployCommand := "kubectl wait --for condition=Available=True deployment -n kube-system metrics-server --timeout=240s"
	waitPodsCommand := "kubectl wait --for condition=ready pod -n kube-system -l k8s-app=metrics-server --timeout=240s"
	command := fmt.Sprintf("%s && %s && %s", installCommand, waitDeployCommand, waitPodsCommand)
	return command, []string{getBinding(hostKubeconfigPath, kubeConfigPath)}
}

func GetInstallHelmChartCommand(name, namespace, hostKubeconfigPath string) (string, []string) {
	command := fmt.Sprintf("helm install %s /resources/testchart --namespace %s --create-namespace", name, namespace)
	return command, []string{getBinding(hostKubeconfigPath, kubeConfigPath)}
}

func GetInstallOsmCommand(hostKubeconfigPath string) (string, []string) {
	// https://release-v1-1.docs.openservicemesh.io/docs/getting_started/setup_osm/
	command := fmt.Sprintf(`osm install \
	--mesh-name %s \
	--set=osm.enablePermissiveTrafficPolicy=false \
	--set=osm.deployPrometheus=true \
	--set=osm.deployGrafana=true \
	--set=osm.deployJaeger=true`, osmName)

	return command, []string{getBinding(hostKubeconfigPath, kubeConfigPath)}
}

func GetUninstallOsmCommand(hostKubeconfigPath string) (string, []string) {
	// https://release-v1-1.docs.openservicemesh.io/docs/getting_started/setup_osm/
	command := fmt.Sprintf(`osm uninstall mesh \
	--mesh-name %s \
	--force \
	--delete-cluster-wide-resources`, osmName)
	return command, []string{getBinding(hostKubeconfigPath, kubeConfigPath)}
}

func GetAddOsmNamespacesCommand(hostKubeconfigPath string) (string, []string) {
	command := fmt.Sprintf("osm namespace add bookstore bookbuyer bookthief bookwarehouse --mesh-name %s", osmName)
	return command, []string{getBinding(hostKubeconfigPath, kubeConfigPath)}
}

func GetDeployOsmAppsCommand(hostKubeconfigPath string) (string, []string) {
	commands := []string{
		"kubectl apply -f /resources/osm-apps/bookbuyer.yaml",
		"kubectl apply -f /resources/osm-apps/bookthief.yaml",
		"kubectl apply -f /resources/osm-apps/bookstore.yaml",
		"kubectl apply -f /resources/osm-apps/bookstore-v2.yaml",
		"kubectl apply -f /resources/osm-apps/bookwarehouse.yaml",
		"kubectl apply -f /resources/osm-apps/mysql.yaml",
		"kubectl apply -f /resources/osm-apps/traffic-access.yaml",
		"kubectl apply -f /resources/osm-apps/traffic-split.yaml",
	}

	return strings.Join(commands, " && "), []string{getBinding(hostKubeconfigPath, kubeConfigPath)}
}

func getBinding(hostPath, containerPath string) string {
	if runtime.GOOS == "windows" {
		hostPath = strings.Replace(filepath.ToSlash(hostPath), "C:", "/c", 1)
	}

	return fmt.Sprintf("%s:%s", hostPath, containerPath)
}
