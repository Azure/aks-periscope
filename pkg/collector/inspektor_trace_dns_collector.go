package collector

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"github.com/cilium/ebpf/rlimit"
	containercollection "github.com/inspektor-gadget/inspektor-gadget/pkg/container-collection"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/container-collection/networktracer"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/dns/tracer"
	dnstypes "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/dns/types"
	eventtypes "github.com/inspektor-gadget/inspektor-gadget/pkg/types"

	restclient "k8s.io/client-go/rest"
)

// InspektorGadgetDNSTraceCollector defines a InspektorGadget Trace DNS Collector struct
type InspektorGadgetDNSTraceCollector struct {
	data                       map[string]string
	osIdentifier               utils.OSIdentifier
	kubeconfig                 *restclient.Config
	runtimeInfo                *utils.RuntimeInfo
	waiter                     func()
	containerCollectionOptions []containercollection.ContainerCollectionOption
}

// CheckSupported implements the interface method
func (collector *InspektorGadgetDNSTraceCollector) CheckSupported() error {
	// Inspektor Gadget relies on eBPF which is not (currently) available on Windows nodes.
	if collector.osIdentifier != utils.Linux {
		return fmt.Errorf("unsupported OS: %s", collector.osIdentifier)
	}
	return nil
}

// NewInspektorGadgetDNSTraceCollector is a constructor.
func NewInspektorGadgetDNSTraceCollector(
	osIdentifier utils.OSIdentifier,
	config *restclient.Config,
	runtimeInfo *utils.RuntimeInfo,
	waiter func(),
	containerCollectionOptions []containercollection.ContainerCollectionOption,
) *InspektorGadgetDNSTraceCollector {
	return &InspektorGadgetDNSTraceCollector{
		data:                       make(map[string]string),
		osIdentifier:               osIdentifier,
		kubeconfig:                 config,
		runtimeInfo:                runtimeInfo,
		waiter:                     waiter,
		containerCollectionOptions: containerCollectionOptions,
	}
}

func (collector *InspektorGadgetDNSTraceCollector) GetName() string {
	return "inspektorgadget-dns"
}

// Collect implements the interface method
func (collector *InspektorGadgetDNSTraceCollector) Collect() error {
	// From https://www.inspektor-gadget.io/blog/2022/09/using-inspektor-gadget-from-golang-applications/
	// In some kernel versions it's needed to bump the rlimits to
	// use run BPF programs.
	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("failed to remove memlock: %w", err)
	}

	tracer, err := tracer.NewTracer()
	if err != nil {
		return fmt.Errorf("failed to start dns tracer: %w", err)
	}
	defer tracer.Close()

	nodeName := collector.runtimeInfo.HostNodeName

	var mu sync.Mutex
	events := []string{}

	eventCallback := func(container *containercollection.Container, event dnstypes.Event) {
		// Enrich event with data from container
		event.Node = nodeName
		if !container.HostNetwork {
			event.Namespace = container.Namespace
			event.Pod = container.Podname
			event.Container = container.Name
		}

		eventString := eventtypes.EventString(event)

		mu.Lock()
		defer mu.Unlock()
		events = append(events, eventString)
	}

	callback := func(event containercollection.PubSubEvent) {
		// This doesn't *do* anything, but there will be runtime errors if we don't supply a callback.
		log.Printf("Container event %q:\n\t%s", event.Type, eventtypes.EventString(event.Container))
	}

	opts := append(collector.containerCollectionOptions,
		containercollection.WithPubSub(callback),
		containercollection.WithNodeName(nodeName),
		containercollection.WithCgroupEnrichment(),
		containercollection.WithLinuxNamespaceEnrichment(),
		containercollection.WithKubernetesEnrichment(nodeName, collector.kubeconfig),
	)

	containerCollection := &containercollection.ContainerCollection{}
	if err = containerCollection.Initialize(opts...); err != nil {
		return fmt.Errorf("failed to initialize container collection: %w", err)
	}
	defer containerCollection.Close()

	config := &networktracer.ConnectToContainerCollectionConfig[dnstypes.Event]{
		Tracer:        tracer,
		Resolver:      containerCollection,
		Selector:      containercollection.ContainerSelector{},
		EventCallback: eventCallback,
		Base:          dnstypes.Base,
	}

	conn, err := networktracer.ConnectToContainerCollection(config)
	if err != nil {
		return fmt.Errorf("failed to connect network tracer: %w", err)
	}
	defer conn.Close()

	collector.waiter()

	mu.Lock()
	defer mu.Unlock()
	collector.data["dnstracer"] = strings.Join(events, "\n")
	log.Print(collector.data["dnstracer"])

	return nil
}

// GetData implements the interface method
func (collector *InspektorGadgetDNSTraceCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}
