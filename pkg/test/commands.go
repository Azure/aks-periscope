package test

import "fmt"

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
	return command, []string{
		fmt.Sprintf("%s:%s", hostKubeconfigPath, kubeConfigPath),
	}
}

func GetInstallHelmChartCommand(name, namespace, hostKubeconfigPath string) (string, []string) {
	command := fmt.Sprintf("helm install %s /resources/testchart --namespace %s --create-namespace", name, namespace)
	return command, []string{
		fmt.Sprintf("%s:%s", hostKubeconfigPath, kubeConfigPath),
	}
}

func GetInstallOsmCommand(hostKubeconfigPath string) (string, []string) {
	// https://release-v1-1.docs.openservicemesh.io/docs/getting_started/setup_osm/
	command := fmt.Sprintf(`osm install \
	--mesh-name %s \
	--set=osm.enablePermissiveTrafficPolicy=true \
	--set=osm.deployPrometheus=true \
	--set=osm.deployGrafana=true \
	--set=osm.deployJaeger=true`, osmName)

	return command, []string{
		fmt.Sprintf("%s:%s", hostKubeconfigPath, kubeConfigPath),
	}
}

func GetUninstallOsmCommand(hostKubeconfigPath string) (string, []string) {
	// https://release-v1-1.docs.openservicemesh.io/docs/getting_started/setup_osm/
	command := fmt.Sprintf(`osm uninstall mesh \
	--mesh-name %s \
	--force \
	--delete-cluster-wide-resources`, osmName)
	return command, []string{
		fmt.Sprintf("%s:%s", hostKubeconfigPath, kubeConfigPath),
	}
}
