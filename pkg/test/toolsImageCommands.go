package test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

func getCreateClusterCommand() string {
	// Create the cluster if it doesn't exist, and output the kubeconfig content
	existsClusterCommand := fmt.Sprintf("kind get clusters | grep -q '^%s$'", testClusterName)
	createClusterCommand := fmt.Sprintf("kind create cluster --name %s --config=/resources/kind-config/config.yaml --wait 5m --image kindest/node:%s", testClusterName, kindNodeTag)
	getKubeConfigCommand := fmt.Sprintf("kind get kubeconfig --name %s", testClusterName)
	return fmt.Sprintf("%s || %s && %s", existsClusterCommand, createClusterCommand, getKubeConfigCommand)
}

func getListNodesCommand() string {
	return fmt.Sprintf("kind get nodes --name %s", testClusterName)
}

func getLoadDockerImagesCommand(images, nodes []string) string {
	return fmt.Sprintf(`echo "%s" | xargs -P8 -n1 kind load docker-image --name %s --nodes %s`, strings.Join(images, " "), testClusterName, strings.Join(nodes, ","))
}

func getInstallMetricsServerCommand(hostKubeconfigPath string) (string, []string) {
	installCommand := "kubectl apply -f /resources/metrics-server/components.yaml"
	waitCommand := "kubectl rollout status -n kube-system deploy/metrics-server --timeout=240s"
	command := fmt.Sprintf("%s && %s", installCommand, waitCommand)
	return command, []string{getKubeConfigBinding(hostKubeconfigPath)}
}

func getInstallOsmCommand(hostKubeconfigPath, namespace string) (string, []string) {
	// https://release-v1-1.docs.openservicemesh.io/docs/guides/install/#helm-install
	// Setting the release name is *supposed* to set the mesh name, but the CLI does not detect this,
	// so it's set again as a command-line override.
	command := fmt.Sprintf(
		"helm install %s osm --repo https://openservicemesh.github.io/osm --version %s --namespace %s --wait --values /resources/osm-config/override.yaml --set osm.meshName=%s",
		meshName, osmVersion, namespace, meshName)

	return command, []string{getKubeConfigBinding(hostKubeconfigPath)}
}

func getUninstallHelmReleasesCommand(hostKubeconfigPath string) (string, []string) {
	// - List all helm releases in all namespaces
	// - Extract release name and namespace from the output
	// - Run helm uninstall for each release and namespace
	command := "helm ls -A | awk '(NR>1){print $1, $2}' | xargs -n2 --no-run-if-empty sh -c 'helm uninstall $0 --namespace $1 --wait'"
	return command, []string{getKubeConfigBinding(hostKubeconfigPath)}
}

func getAddOsmNamespacesCommand(hostKubeconfigPath string, knownNamespaces *KnownNamespaces) (string, []string) {
	command := fmt.Sprintf("osm namespace add %s %s %s %s --mesh-name %s",
		knownNamespaces.OsmBookBuyer, knownNamespaces.OsmBookStore, knownNamespaces.OsmBookThief, knownNamespaces.OsmBookWarehouse,
		meshName)
	return command, []string{getKubeConfigBinding(hostKubeconfigPath)}
}

func getDeployOsmAppsCommand(hostKubeconfigPath string, knownNamespaces *KnownNamespaces) (string, []string) {
	commands := []string{
		fmt.Sprintf("export BOOKBUYER_NS=%s BOOKSTORE_NS=%s BOOKTHIEF_NS=%s BOOKWAREHOUSE_NS=%s",
			knownNamespaces.OsmBookBuyer, knownNamespaces.OsmBookStore, knownNamespaces.OsmBookThief, knownNamespaces.OsmBookWarehouse),
		"cat /resources/osm-apps/bookbuyer.yaml | envsubst | kubectl apply -f -",
		"cat /resources/osm-apps/bookthief.yaml | envsubst | kubectl apply -f -",
		"cat /resources/osm-apps/bookstore.yaml | envsubst | kubectl apply -f -",
		"cat /resources/osm-apps/bookstore-v2.yaml | envsubst | kubectl apply -f -",
		"cat /resources/osm-apps/bookwarehouse.yaml | envsubst | kubectl apply -f -",
		"cat /resources/osm-apps/mysql.yaml | envsubst | kubectl apply -f -",
		"cat /resources/osm-apps/traffic-access.yaml | envsubst | kubectl apply -f -",
		"cat /resources/osm-apps/traffic-split.yaml | envsubst | kubectl apply -f -",
	}

	return strings.Join(commands, " && "), []string{getKubeConfigBinding(hostKubeconfigPath)}
}

func getTestDiagnosticsCommand(hostKubeconfigPath string) (string, []string) {
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

	return strings.Join(commands, " && "), []string{getKubeConfigBinding(hostKubeconfigPath)}
}

func getKubeConfigBinding(hostKubeconfigPath string) string {
	return getBinding(hostKubeconfigPath, kubeConfigPath)
}

func getBinding(hostPath, containerPath string) string {
	if runtime.GOOS == "windows" {
		hostPath = strings.Replace(filepath.ToSlash(hostPath), "C:", "/c", 1)
	}

	return fmt.Sprintf("%s:%s", hostPath, containerPath)
}
