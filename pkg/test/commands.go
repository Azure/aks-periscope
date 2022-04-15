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
