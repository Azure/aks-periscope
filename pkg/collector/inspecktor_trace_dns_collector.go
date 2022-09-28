package collector

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	gadgetv1alpha1 "github.com/kinvolk/inspektor-gadget/pkg/apis/gadget/v1alpha1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
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
	err = collector.runTraceCommandOnPod(gadgetName)
	if err != nil {
		return err
	}

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

func (collector *InspektorGadgetDNSTraceCollector) runTraceCommandOnPod(gadgetName string) error {
	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(collector.kubeconfig)
	if err != nil {
		return fmt.Errorf("getting access to K8S failed: %w", err)
	}

	gadgetPods, err := clientset.CoreV1().Pods("gadget").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list gadget pods: %w", err)
	}
	var command = []string{"./bin/gadgettracermanager", "-call", "receive-stream", "-tracerid", fmt.Sprintf("trace_gadget_%s", gadgetName)}

	for _, podName := range gadgetPods.Items {
		request := clientset.CoreV1().RESTClient().Post().
			Resource("pods").
			Name(podName.Name).
			Namespace("gadget").
			SubResource("exec").
			VersionedParams(&v1.PodExecOptions{
				Stdin:   false,
				Stdout:  true,
				Stderr:  true,
				TTY:     false,
				Command: command,
			}, scheme.ParameterCodec)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		streamOptions := remotecommand.StreamOptions{
			Stdout: stdout,
			Stderr: stderr,
		}

		exec, err := remotecommand.NewSPDYExecutor(collector.kubeconfig, "POST", request.URL())
		log.Printf("\tPost request to DNS trace stream : %s ", request.URL())
		if err != nil {
			return fmt.Errorf("could not create SPDY executor: %w", err)
		}

		err = exec.Stream(streamOptions)
		if err != nil {
			result := strings.TrimSpace(stdout.String()) + "\n" + strings.TrimSpace(stderr.String())
			result = strings.TrimSpace(result)
			collector.data[fmt.Sprintf("dns-trace-%s-%s", gadgetName, podName)] = result
			return fmt.Errorf("error reading stream: %w", err)
		}
		result := strings.TrimSpace(stdout.String()) + "\n" + strings.TrimSpace(stderr.String())
		result = strings.TrimSpace(result)
		log.Printf("\tObtain DNS trace stream : %s ", result)

		collector.data[fmt.Sprintf("dns-trace-%s-%s", gadgetName, podName)] = result
		log.Printf("\tupdated collector data")
	}

	return nil
}