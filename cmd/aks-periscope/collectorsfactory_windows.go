package main

import (
	"github.com/Azure/aks-periscope/pkg/interfaces"
)

func addOSSpecificCollectors(collectors []interfaces.Collector, config *restclient.Config, runtimeInfo *utils.RuntimeInfo) []interfaces.Collector {
	return collectors
}
