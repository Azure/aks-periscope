package gadget

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"github.com/cilium/ebpf/rlimit"
	restclient "k8s.io/client-go/rest"
)

const (
	GadgetOperation = "gadget.kinvolk.io/operation"
)

// InspektorGadgetTraceCollector defines a InspektorGadget Trace Collector that are common to trace gadgets
type InspektorGadgetTraceCollector struct {
	Data             map[string]string
	OsIdentifier     utils.OSIdentifier
	Kubeconfig       *restclient.Config
	CommandRunner    *utils.KubeCommandRunner
	RuntimeInfo      *utils.RuntimeInfo
	CollectingPeriod time.Duration
}

type Tracer interface {
	Stop()
}

func (collector *InspektorGadgetTraceCollector) CheckSupported() error {
	// Inspektor Gadget relies on eBPF which is not (currently) available on Windows nodes.
	if collector.OsIdentifier != utils.Linux {
		return fmt.Errorf("unsupported OS: %s", collector.OsIdentifier)
	}

	crds, err := collector.CommandRunner.GetCRDUnstructuredList()
	if err != nil {
		return fmt.Errorf("error listing CRDs in cluster")
	}

	for _, crd := range crds.Items {
		if strings.Contains(crd.GetName(), "traces.gadget.kinvolk.io") {
			return nil
		}
	}
	return fmt.Errorf("does not contain gadget crd")
}

func (collector *InspektorGadgetTraceCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.Data)
}

func (collector *InspektorGadgetTraceCollector) Collect(gadgetName string, tracer Tracer) error {
	// From https://www.inspektor-gadget.io/blog/2022/09/using-inspektor-gadget-from-golang-applications/
	// In some kernel versions it's needed to bump the rlimits to
	// use run BPF programs.
	if err := rlimit.RemoveMemlock(); err != nil {
		// Well...maybe we can continue anyway? No harm in trying. Log the error and continue.
		log.Printf("\tcould not remove memlock: %v", err)
	}

	defer tracer.Stop()

	log.Printf("\twait for %v to stop collection", collector.CollectingPeriod)
	time.Sleep(collector.CollectingPeriod)

	return nil
}
