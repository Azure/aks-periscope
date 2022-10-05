package inspektor_gadget

import (
	"time"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	restclient "k8s.io/client-go/rest"
)

// InspektorGadgetTCPTraceCollector defines a InspektorGadget Trace TCP Collector struct
type InspektorGadgetTCPTraceCollector struct {
	tracerGadget *InspektorGadgetTraceCollector
}

func (collector *InspektorGadgetTCPTraceCollector) CheckSupported() error {
	return collector.tracerGadget.CheckSupported()
}

// NewInspektorGadgetTCPTraceCollector is a constructor.
func NewInspektorGadgetTCPTraceCollector(config *restclient.Config, runtimeInfo *utils.RuntimeInfo) *InspektorGadgetTCPTraceCollector {

	return &InspektorGadgetTCPTraceCollector{
		tracerGadget: &InspektorGadgetTraceCollector{
			data:          make(map[string]string),
			kubeconfig:    config,
			commandRunner: utils.NewKubeCommandRunner(config),
			runtimeInfo:   runtimeInfo,
		},
	}
}

func (collector *InspektorGadgetTCPTraceCollector) GetName() string {
	return "inspektorgadget-tcptrace"
}

// Collect implements the interface method
func (collector *InspektorGadgetTCPTraceCollector) Collect() error {
	return collector.tracerGadget.collect("tcptracer", 2*time.Minute)
}

func (collector *InspektorGadgetTCPTraceCollector) GetData() map[string]interfaces.DataValue {
	return collector.tracerGadget.GetData()
}
