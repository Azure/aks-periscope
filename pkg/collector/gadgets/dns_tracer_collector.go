package gadgets

import (
	"fmt"
	"github.com/cilium/ebpf/rlimit"
	"log"
	"time"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/dns/tracer"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/dns/types"
	"os"
)

// DNSTracerCollector defines a DNS tracer Collector struct
type DNSTracerCollector struct {
	data         map[string]string
	osIdentifier utils.OSIdentifier
}

// NewDNSTracerCollector is a constructor to collect DNS trace data using IG
func NewDNSTracerCollector(osIdentifier utils.OSIdentifier) *DNSTracerCollector {
	return &DNSTracerCollector{
		osIdentifier: osIdentifier,
		data:         make(map[string]string),
	}
}

func (collector *DNSTracerCollector) GetName() string {
	return "dns-tracer"
}

func (collector *DNSTracerCollector) CheckSupported() error {
	// check for OS since ebpf is only available on linux
	if collector.osIdentifier != "linux" {
		return fmt.Errorf("tracer uses ebpf capabilities which is supported on linux only")
	}
	// In some kernel versions it's needed to bump the rlimits to use run BPF programs.
	if err := rlimit.RemoveMemlock(); err != nil {
		return err
	}

	return nil
}

// Collect implements the interface method
func (collector *DNSTracerCollector) Collect() error {

	// Define a callback to be called each time there is an event, capture in the collector data
	eventCallback := func(event types.Event) {
		//PktType    string     `json:"pktType,omitempty" column:"type,minWidth:7,maxWidth:9"`
		qr := event.Qr
		if qr == types.DNSPktTypeQuery {
			qr = "request"
		} else if qr == types.DNSPktTypeResponse {
			qr = "response"
		}
		result := fmt.Sprintf("A new %q dns %s about %s using packet type %s was observed. nameserver: %s, response: %s\n",
			event.QType, qr, event.DNSName, event.PktType, event.Nameserver, event.Rcode)
		log.Printf(result)
		collector.data[fmt.Sprintf("%s-%s", collector.GetName(), event.ID)] = result
	}

	// Create tracer. In this case no parameters are passed.
	dnsTracer, err := tracer.NewTracer()
	if err != nil {
		return fmt.Errorf("error creating tracer: %s\n", err)
	}
	defer dnsTracer.Close()

	pid := uint32(os.Getpid())
	log.Printf("attach tracer to pid %v", pid)
	if err := dnsTracer.Attach(pid, eventCallback); err != nil {
		return fmt.Errorf("error attaching tracer: %v\n", err)
	}

	defer func() {
		if err := dnsTracer.Detach(pid); err != nil {
			log.Fatalf("Could not detach tracer %v", err)
		}
	}()

	//capture the result for 30 seconds.
	done := make(chan struct{})
	go func() {
		time.Sleep(30 * time.Second)
		close(done)
	}()

	<-done
	return nil
}

func (collector *DNSTracerCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}
