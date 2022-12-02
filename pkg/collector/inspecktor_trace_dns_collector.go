package collector

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

// CheckSupported implements the interface method
func (collector *InspektorGadgetDNSTraceCollector) CheckSupported() error {
	return collector.tracerGadget.CheckSupported()
}

// NewInspektorGadgetDNSTraceCollector is a constructor.
func NewInspektorGadgetDNSTraceCollector(osIdentifier utils.OSIdentifier, config *restclient.Config, runtimeInfo *utils.RuntimeInfo, collectingPeriod time.Duration) *InspektorGadgetDNSTraceCollector {

	return &InspektorGadgetDNSTraceCollector{
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

func (collector *InspektorGadgetDNSTraceCollector) GetName() string {
	return "inspektorgadget-dns"
}

// Collect implements the interface method
func (collector *InspektorGadgetDNSTraceCollector) Collect() error {
	return collector.tracerGadget.collect("dns")
}

// GetData implements the interface method
func (collector *InspektorGadgetDNSTraceCollector) GetData() map[string]interfaces.DataValue {
	return collector.tracerGadget.GetData()
}
