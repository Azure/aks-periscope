package collector

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	gadgetv1alpha1 "github.com/kinvolk/inspektor-gadget/pkg/apis/gadget/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	restclient "k8s.io/client-go/rest"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	GadgetOperation = "gadget.kinvolk.io/operation"
)

// InspektorGadgetDNSTraceCollector defines a InspektorGadget Trace DNS Collector struct
type InspektorGadgetDNSTraceCollector struct {
	data          map[string]string
	kubeconfig    *restclient.Config
	commandRunner *utils.KubeCommandRunner
	runtimeInfo   *utils.RuntimeInfo
}

// NewInspektorGadgetDNSTraceCollector is a constructor.
func NewInspektorGadgetDNSTraceCollector(config *restclient.Config, runtimeInfo *utils.RuntimeInfo) *InspektorGadgetDNSTraceCollector {
	rand.Seed(time.Now().UnixNano())
	return &InspektorGadgetDNSTraceCollector{
		data:          make(map[string]string),
		kubeconfig:    config,
		commandRunner: utils.NewKubeCommandRunner(config),
		runtimeInfo:   runtimeInfo,
	}
}

func (collector *InspektorGadgetDNSTraceCollector) GetName() string {
	return "inspektorgadget-dnstrace"
}

func (collector *InspektorGadgetDNSTraceCollector) CheckSupported() error {
	crds, err := collector.commandRunner.GetCRDUnstructuredList()
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

// Collect implements the interface method
func (collector *InspektorGadgetDNSTraceCollector) Collect() error {

	gadgetScheme := runtime.NewScheme()

	err := gadgetv1alpha1.AddToScheme(gadgetScheme)
	if err != nil {
		return fmt.Errorf("could not add gadget scheme: %w", err)
	}

	gadgetClient, err := runtimeclient.New(collector.kubeconfig, runtimeclient.Options{
		Scheme: gadgetScheme,
	})
	if err != nil {
		return fmt.Errorf("could not create rest client for gadgets: %w", err)
	}

	// Create a dns gadget.
	gadgetName := fmt.Sprintf("dns-%s", randomPodID())
	trace := &gadgetv1alpha1.Trace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "gadget",
			Annotations: map[string]string{
				GadgetOperation: string(gadgetv1alpha1.OperationStart),
			},
			Name: gadgetName,
		},
		Spec: gadgetv1alpha1.TraceSpec{
			Node:       collector.runtimeInfo.HostNodeName,
			Gadget:     "dns",
			RunMode:    gadgetv1alpha1.RunModeManual,
			OutputMode: gadgetv1alpha1.TraceOutputModeStream,
		},
	}
	err = gadgetClient.Create(context.TODO(), trace)
	if err != nil {
		return fmt.Errorf("could not create dns trace %s: %w", gadgetName, err)
	}
	//collect output
	output, err := utils.RunCommandOnHost("./bin/gadgettracermanager", "-call", "receive-stream", "-tracerid", fmt.Sprintf("trace_%s_%s", "gadget", gadgetName))
	if err != nil {
		return err
	}
	collector.data[fmt.Sprintf("dns-trace-%s", gadgetName)] = output

	return nil
}

func (collector *InspektorGadgetDNSTraceCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}

func randomPodID() string {
	output := make([]byte, 5)
	allowedCharacters := "0123456789abcdefghijklmnopqrstuvwxyz"
	for i := range output {
		output[i] = allowedCharacters[rand.Int31n(int32(len(allowedCharacters)))]
	}
	return string(output)
}
