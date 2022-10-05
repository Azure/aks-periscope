package inspektor_gadget

import (
	"time"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	restclient "k8s.io/client-go/rest"
)

// InspektorGadgetDNSTraceCollector defines a InspektorGadget Trace DNS Collector struct
type InspektorGadgetDNSTraceCollector struct {
	tracerGadget *InspektorGadgetTraceCollector
}

func (collector *InspektorGadgetDNSTraceCollector) Collect() error {
	return collector.tracerGadget.collect("dns", 2*time.Minute)
}

func (collector *InspektorGadgetDNSTraceCollector) GetData() map[string]interfaces.DataValue {
	//TODO implement me
	panic("implement me")
}

func (collector *InspektorGadgetDNSTraceCollector) CheckSupported() error {
	return collector.tracerGadget.CheckSupported()
}

// NewInspektorGadgetDNSTraceCollector is a constructor.
func NewInspektorGadgetDNSTraceCollector(config *restclient.Config, runtimeInfo *utils.RuntimeInfo) *InspektorGadgetDNSTraceCollector {
	return &InspektorGadgetDNSTraceCollector{
		tracerGadget: &InspektorGadgetTraceCollector{
			data:          make(map[string]string),
			kubeconfig:    config,
			commandRunner: utils.NewKubeCommandRunner(config),
			runtimeInfo:   runtimeInfo,
		},
	}
}

func (collector *InspektorGadgetDNSTraceCollector) GetName() string {
	return "inspektorgadget-dnstrace"
}
