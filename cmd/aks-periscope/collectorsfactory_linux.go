package main

import (
	"log"
	"time"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"

	containercollection "github.com/inspektor-gadget/inspektor-gadget/pkg/container-collection"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/runcfanotify"

	restclient "k8s.io/client-go/rest"
)

func addOSSpecificCollectors(collectors []interfaces.Collector, config *restclient.Config, runtimeInfo *utils.RuntimeInfo) []interfaces.Collector {
	// Use the default InspektorGadget behaviour for determining containers:
	// https://github.com/inspektor-gadget/inspektor-gadget/blob/6b00fea3f925c9da478126931e774e340ca9bfdf/pkg/gadgettracermanager/gadgettracermanager.go#L275-L283
	var containerCollectionOptions []containercollection.ContainerCollectionOption
	if runcfanotify.Supported() {
		containerCollectionOptions = []containercollection.ContainerCollectionOption{
			containercollection.WithRuncFanotify(),
			containercollection.WithInitialKubernetesContainers(runtimeInfo.HostNodeName),
		}
	} else {
		containerCollectionOptions = []containercollection.ContainerCollectionOption{
			containercollection.WithPodInformer(runtimeInfo.HostNodeName),
		}
	}

	containerCollectionOptions = append(
		containerCollectionOptions,
		containercollection.WithNodeName(runtimeInfo.HostNodeName),
		containercollection.WithCgroupEnrichment(),
		containercollection.WithLinuxNamespaceEnrichment(),
		containercollection.WithKubernetesEnrichment(runtimeInfo.HostNodeName, config),
	)

	// Traces can produce a lot of data.
	// TODO: Consider whether this should be lower or configurable.
	traceCollectionPeriod := 30 * time.Second
	traceWaiter := func() {
		log.Printf("\twait for %v to stop collection", traceCollectionPeriod)
		time.Sleep(traceCollectionPeriod)
	}

	return append(collectors,
		collector.NewInspektorGadgetDNSTraceCollector(runtimeInfo, traceWaiter, containerCollectionOptions),
		collector.NewInspektorGadgetTCPTraceCollector(runtimeInfo, traceWaiter, containerCollectionOptions),
	)
}
