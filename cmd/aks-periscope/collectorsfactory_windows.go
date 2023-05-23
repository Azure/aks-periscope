package main

import (
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	restclient "k8s.io/client-go/rest"
)

func addOSSpecificCollectors(collectors []interfaces.Collector, config *restclient.Config, runtimeInfo *utils.RuntimeInfo) []interfaces.Collector {
	return collectors
}
