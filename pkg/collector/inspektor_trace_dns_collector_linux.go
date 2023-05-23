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
)

// InspektorGadgetDNSTraceCollector defines a InspektorGadget Trace DNS Collector struct
type InspektorGadgetDNSTraceCollector struct {
	data                       map[string]string
	runtimeInfo                *utils.RuntimeInfo
	waiter                     func()
	containerCollectionOptions []containercollection.ContainerCollectionOption
}

// CheckSupported implements the interface method
func (collector *InspektorGadgetDNSTraceCollector) CheckSupported() error {
	// Inspektor Gadget relies on eBPF which is not (currently) available on Windows nodes.
	// However, we're only compiling this for Linux OS right now, so we can skip the OS check.
	return nil
}

// NewInspektorGadgetDNSTraceCollector is a constructor.
func NewInspektorGadgetDNSTraceCollector(
	runtimeInfo *utils.RuntimeInfo,
	waiter func(),
	containerCollectionOptions []containercollection.ContainerCollectionOption,
) *InspektorGadgetDNSTraceCollector {
	return &InspektorGadgetDNSTraceCollector{
		data:                       make(map[string]string),
		runtimeInfo:                runtimeInfo,
		waiter:                     waiter,
		containerCollectionOptions: containerCollectionOptions,
	}
}

func (collector *InspektorGadgetDNSTraceCollector) GetName() string {
	return "inspektorgadget-dnstrace"
}

// Collect implements the interface method
func (collector *InspektorGadgetDNSTraceCollector) Collect() error {
	// From https://www.inspektor-gadget.io/blog/2022/09/using-inspektor-gadget-from-golang-applications/
	// In some kernel versions it's needed to bump the rlimits to
	// use run BPF programs.
	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("failed to remove memlock: %w", err)
	}

	// We want to trace DNS queries from all pods running on the node, not just the current process.
	// To do this we need to make use of a ContainerCollection, which can be initially populated
	// with all the pod processes, and dynamically updated as pods are created and deleted.
	containerEventCallback := func(event containercollection.PubSubEvent) {
		// This doesn't *do* anything, but there will be runtime errors if we don't supply a callback.
		switch event.Type {
		case containercollection.EventTypeAddContainer:
			log.Printf("Container added: %q pid %d\n", event.Container.Name, event.Container.Pid)
		case containercollection.EventTypeRemoveContainer:
			log.Printf("Container removed: %q pid %d\n", event.Container.Name, event.Container.Pid)
		}
	}

	// Use the supplied container collection options, but prepend the container event callback.
	// The options are all functions that are executed when the container collection is initialized.
	opts := append(
		[]containercollection.ContainerCollectionOption{containercollection.WithPubSub(containerEventCallback)},
		collector.containerCollectionOptions...,
	)

	// Initialize the container collection
	containerCollection := &containercollection.ContainerCollection{}
	if err := containerCollection.Initialize(opts...); err != nil {
		return fmt.Errorf("failed to initialize container collection: %w", err)
	}
	defer containerCollection.Close()

	// Build up a collection of DNS query events, with a mutex to protect against concurrent access.
	var mu sync.Mutex
	events := []string{}

	// Events will be collected in a callback from the DNS tracer.
	dnsEventCallback := func(container *containercollection.Container, event dnstypes.Event) {
		// Enrich event with data from container
		event.Node = collector.runtimeInfo.HostNodeName
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

	// The DNS tracer by itself is not associated with any process. It will need to be 'connected'
	// to the container collection, which will manage the attaching and detaching of PIDs as
	// containers are created and deleted.
	tracer, err := tracer.NewTracer()
	if err != nil {
		return fmt.Errorf("failed to start dns tracer: %w", err)
	}
	defer tracer.Close()

	// Set up the information needed to link the tracer to the containers. The selector is empty,
	// meaning that all containers in the collection will be traced.
	config := &networktracer.ConnectToContainerCollectionConfig[dnstypes.Event]{
		Tracer:        tracer,
		Resolver:      containerCollection,
		Selector:      containercollection.ContainerSelector{},
		EventCallback: dnsEventCallback,
		Base:          dnstypes.Base,
	}

	// Connect the tracer up. Closing the connection will detach the PIDs from the tracer.
	conn, err := networktracer.ConnectToContainerCollection(config)
	if err != nil {
		return fmt.Errorf("failed to connect network tracer: %w", err)
	}
	defer conn.Close()

	// The trace is now running. Run whatever function our consumer has supplied before storing the
	// collected data.
	collector.waiter()

	// Store the collected data.
	func() {
		mu.Lock()
		defer mu.Unlock()
		collector.data["dnstracer"] = strings.Join(events, "\n")
	}()

	return nil
}

// GetData implements the interface method
func (collector *InspektorGadgetDNSTraceCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}
