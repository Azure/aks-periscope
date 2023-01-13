package collector

import (
	"fmt"
	"time"

	"github.com/Azure/aks-periscope/pkg/collector/gadget"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	tcptracer "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/tcp/tracer"
	tcptypes "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/tcp/types"
	restclient "k8s.io/client-go/rest"
)

// InspektorGadgetTCPTraceCollector defines a InspektorGadget Trace TCP Collector struct
type InspektorGadgetTCPTraceCollector struct {
	tracerGadget *gadget.InspektorGadgetTraceCollector
}

// CheckSupported implements the interface method
func (collector *InspektorGadgetTCPTraceCollector) CheckSupported() error {
	return collector.tracerGadget.CheckSupported()
}

// NewInspektorGadgetTCPTraceCollector is a constructor.
func NewInspektorGadgetTCPTraceCollector(osIdentifier utils.OSIdentifier, config *restclient.Config, runtimeInfo *utils.RuntimeInfo, collectingPeriod time.Duration) *InspektorGadgetTCPTraceCollector {
	return &InspektorGadgetTCPTraceCollector{
		tracerGadget: &gadget.InspektorGadgetTraceCollector{
			Data:             make(map[string]string),
			OsIdentifier:     osIdentifier,
			Kubeconfig:       config,
			CommandRunner:    utils.NewKubeCommandRunner(config),
			RuntimeInfo:      runtimeInfo,
			CollectingPeriod: collectingPeriod,
		},
	}
}

func (collector *InspektorGadgetTCPTraceCollector) GetName() string {
	return "inspektorgadget-tcptrace"
}

// Collect implements the interface method
func (collector *InspektorGadgetTCPTraceCollector) Collect() error {
	eventCallback := func(event tcptypes.Event) {
		collector.tracerGadget.Data["tcptracer"] = fmt.Sprintf("A new %q process with pid %d was executed\n", event.Comm, event.Pid)
	}

	tracer, err := tcptracer.NewTracer(&tcptracer.Config{}, nil, eventCallback)
	if err != nil {
		return fmt.Errorf("could not create tracer: %w", err)
	}

	return collector.tracerGadget.Collect("tcptracer", tracer)
}

// GetData implements the interface method
func (collector *InspektorGadgetTCPTraceCollector) GetData() map[string]interfaces.DataValue {
	return collector.tracerGadget.GetData()
}
