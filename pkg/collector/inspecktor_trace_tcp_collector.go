package collector

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

// CheckSupported implements the interface method
func (collector *InspektorGadgetTCPTraceCollector) CheckSupported() error {
	return collector.tracerGadget.CheckSupported()
}

// NewInspektorGadgetTCPTraceCollector is a constructor.
func NewInspektorGadgetTCPTraceCollector(osIdentifier utils.OSIdentifier, config *restclient.Config, runtimeInfo *utils.RuntimeInfo, collectingPeriod time.Duration) *InspektorGadgetTCPTraceCollector {

	return &InspektorGadgetTCPTraceCollector{
		tracerGadget: &InspektorGadgetTraceCollector{
			data:             make(map[string]string),
			osIdentifier:     osIdentifier,
			kubeconfig:       config,
			commandRunner:    utils.NewKubeCommandRunner(config),
			runtimeInfo:      runtimeInfo,
			collectingPeriod: collectingPeriod,
		},
	}
}

func (collector *InspektorGadgetTCPTraceCollector) GetName() string {
	return "inspektorgadget-tcptrace"
}

// Collect implements the interface method
func (collector *InspektorGadgetTCPTraceCollector) Collect() error {
	return collector.tracerGadget.collect("tcptracer")
}

// GetData implements the interface method
func (collector *InspektorGadgetTCPTraceCollector) GetData() map[string]interfaces.DataValue {
	return collector.tracerGadget.GetData()
}
