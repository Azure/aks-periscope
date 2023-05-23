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
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-collection/gadgets/trace"
	tcptracer "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/tcp/tracer"
	tcptypes "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/tcp/types"
	standardtracer "github.com/inspektor-gadget/inspektor-gadget/pkg/standardgadgets/trace/tcp"
	eventtypes "github.com/inspektor-gadget/inspektor-gadget/pkg/types"
)

// InspektorGadgetTCPTraceCollector defines a InspektorGadget Trace TCP Collector struct
type InspektorGadgetTCPTraceCollector struct {
	data                       map[string]string
	runtimeInfo                *utils.RuntimeInfo
	waiter                     func()
	containerCollectionOptions []containercollection.ContainerCollectionOption
}

// CheckSupported implements the interface method
func (collector *InspektorGadgetTCPTraceCollector) CheckSupported() error {
	// Inspektor Gadget relies on eBPF which is not (currently) available on Windows nodes.
	// However, we're only compiling this for Linux OS right now, so we can skip the OS check.
	return nil
}

// NewInspektorGadgetTCPTraceCollector is a constructor.
func NewInspektorGadgetTCPTraceCollector(
	runtimeInfo *utils.RuntimeInfo,
	waiter func(),
	containerCollectionOptions []containercollection.ContainerCollectionOption,
) *InspektorGadgetTCPTraceCollector {
	return &InspektorGadgetTCPTraceCollector{
		data:                       make(map[string]string),
		runtimeInfo:                runtimeInfo,
		waiter:                     waiter,
		containerCollectionOptions: containerCollectionOptions,
	}
}

func (collector *InspektorGadgetTCPTraceCollector) GetName() string {
	return "inspektorgadget-tcptrace"
}

// Collect implements the interface method
func (collector *InspektorGadgetTCPTraceCollector) Collect() error {
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

	tcpEventCallback := func(event tcptypes.Event) {
		eventString := eventtypes.EventString(event)

		mu.Lock()
		defer mu.Unlock()
		events = append(events, eventString)
	}

	traceConfig := &tcptracer.Config{}
	enricher := containerCollection

	var tracer trace.Tracer
	tracer, err := tcptracer.NewTracer(traceConfig, enricher, tcpEventCallback)
	if err != nil {
		log.Printf("Failed to create core tracer, falling back to standard one: %v", err)
		tracer, err = standardtracer.NewTracer(traceConfig, tcpEventCallback)
		if err != nil {
			return fmt.Errorf("failed to create a tracer: %w", err)
		}
	}
	defer tracer.Stop()

	// The trace is now running. Run whatever function our consumer has supplied before storing the
	// collected data.
	collector.waiter()

	// Store the collected data.
	func() {
		mu.Lock()
		defer mu.Unlock()
		collector.data["tcptracer"] = strings.Join(events, "\n")
	}()

	return nil
}

// GetData implements the interface method
func (collector *InspektorGadgetTCPTraceCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}
