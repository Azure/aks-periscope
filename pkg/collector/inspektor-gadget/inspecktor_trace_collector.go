package inspektor_gadget

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"github.com/cilium/ebpf/rlimit"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/exec/tracer"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/exec/types"
	restclient "k8s.io/client-go/rest"
)

const (
	GadgetOperation = "gadget.kinvolk.io/operation"
)

// InspektorGadgetTraceCollector defines a InspektorGadget Trace Collector that are common to trace gadgets
type InspektorGadgetTraceCollector struct {
	data             map[string]string
	osIdentifier     utils.OSIdentifier
	kubeconfig       *restclient.Config
	commandRunner    *utils.KubeCommandRunner
	runtimeInfo      *utils.RuntimeInfo
	collectingPeriod time.Duration
}

// func (collector *InspektorGadgetTraceCollector) runTraceCommandOnPod(gadgetName string, gadgetClient runtimeclient.Client, trace string) error {

// 	// Creates the clientset
// 	clientset, err := kubernetes.NewForConfig(collector.kubeconfig)
// 	if err != nil {
// 		return fmt.Errorf("getting access to K8S failed: %w", err)
// 	}

// 	podName, err := collector.getGadgetPodName(clientset)
// 	if err != nil {
// 		return fmt.Errorf("failed to get gadget pod name: %w", err)
// 	}

// 	traceName := collector.getTraceName(gadgetName)
// 	command := []string{"./bin/gadgettracermanager", "-call", "receive-stream", "-tracerid", fmt.Sprintf("trace_gadget_%s", traceName)}

// 	collectChan := make(chan error)
// 	go func() {
// 		stdout := new(bytes.Buffer)
// 		stderr := new(bytes.Buffer)
// 		streamOptions := remotecommand.StreamOptions{
// 			Stdout: stdout,
// 			Stderr: stderr,
// 		}

// 		request := clientset.CoreV1().RESTClient().Post().
// 			Resource("pods").
// 			Name(podName).
// 			Namespace("gadget").
// 			SubResource("exec").
// 			VersionedParams(&v1.PodExecOptions{
// 				Stdin:   false,
// 				Stdout:  true,
// 				Stderr:  true,
// 				TTY:     false,
// 				Command: command,
// 			}, scheme.ParameterCodec)

// 		log.Printf("\tPost request to trace stream : %s ", request.URL())
// 		exec, err := remotecommand.NewSPDYExecutor(collector.kubeconfig, "POST", request.URL())
// 		if err != nil {
// 			collectChan <- fmt.Errorf("error creating SPDYExecutor for pod exec %q: %w", podName, err)
// 			return
// 		}

// 		log.Printf("\tCollecting trace stream %s from pod %s", traceName, podName)
// 		err = exec.Stream(streamOptions)
// 		if err != nil {
// 			collectChan <- fmt.Errorf("error executing command %q on %s: %w\nOutput:\n%s", command, podName, err, stderr.String())
// 			return
// 		}

// 		log.Printf("\tCollected trace stream %s from pod %s", traceName, podName)
// 		result := strings.TrimSpace(stdout.String()) + "\n" + strings.TrimSpace(stderr.String())

// 		// Prefix the data key with 'gadget' to distinguish it from other collectors (e.g. the 'dns' collector).
// 		// We don't need the node, pod or trace name in the key, because results are output per-node, and there will
// 		// only be one trace for each gadget on each node.
// 		collector.data[fmt.Sprintf("gadget-%s", gadgetName)] = result
// 		collectChan <- nil
// 	}()

// 	//TODO kill in a proper way by apply annotation
// 	log.Printf("\twait for %v to stop collection", collector.collectingPeriod)
// 	time.Sleep(collector.collectingPeriod)

// 	err = gadgetClient.Delete(context.TODO(), trace)
// 	if err != nil {
// 		log.Printf("could not kill trace %s: %v", trace.Name, err)
// 	}

// 	// wait for the final result to be written
// 	return <-collectChan
// }

// getGadgetPodName gets the name of the 'gadget' pod that runs on the same node as this Periscope instance
// (Inspektor Gadget runs as a DaemonSet, so we expect there to be exactly one of these).
// func (collector *InspektorGadgetTraceCollector) getGadgetPodName(clientset *kubernetes.Clientset) (string, error) {
// 	gadgetPods, err := clientset.CoreV1().Pods("gadget").List(context.TODO(), metav1.ListOptions{})
// 	if err != nil {
// 		return "", fmt.Errorf("could not list gadget pods: %w", err)
// 	}

// 	for _, pod := range gadgetPods.Items {
// 		if pod.Spec.NodeName == collector.runtimeInfo.HostNodeName {
// 			return pod.Name, nil
// 		}
// 	}

// 	return "", fmt.Errorf("no gadget pod found on node %q", collector.runtimeInfo.HostNodeName)
// }

// func (collector *InspektorGadgetTraceCollector) getTraceName(gadgetName string) string {
// 	// There should be at most one trace for each gadget running on each node, so the combination of
// 	// gadget name and hostname should be sufficient to uniquely identify this trace.
// 	return fmt.Sprintf("%s-%s", gadgetName, collector.runtimeInfo.HostNodeName)
// }

func (collector *InspektorGadgetTraceCollector) CheckSupported() error {
	// Inspektor Gadget relies on eBPF which is not (currently) available on Windows nodes.
	if collector.osIdentifier != utils.Linux {
		return fmt.Errorf("unsupported OS: %s", collector.osIdentifier)
	}

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

func (collector *InspektorGadgetTraceCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}

func (collector *InspektorGadgetTraceCollector) collect(gadgetName string) error {
	// From https://www.inspektor-gadget.io/blog/2022/09/using-inspektor-gadget-from-golang-applications/
	// In some kernel versions it's needed to bump the rlimits to
	// use run BPF programs.
	if err := rlimit.RemoveMemlock(); err != nil {
		// Well...maybe we can continue anyway? No harm in trying. Log the error and continue.
		log.Printf("\tcould not remove memlock: %v", err)
	}

	eventCallback := func(event types.Event) {
		collector.data[gadgetName] = fmt.Sprintf("A new %q process with pid %d was executed\n", event.Comm, event.Pid)
	}

	tracer, err := tracer.NewTracer(&tracer.Config{}, nil, eventCallback)
	if err != nil {
		return fmt.Errorf("could not create tracer: %w", err)
	}

	defer tracer.Stop()

	log.Printf("\twait for %v to stop collection", collector.collectingPeriod)
	time.Sleep(collector.collectingPeriod)

	return nil

	///////////////////////////////////////////////

	// gadgetScheme := runtime.NewScheme()

	// err := gadgetv1alpha1.AddToScheme(gadgetScheme)
	// if err != nil {
	// 	return fmt.Errorf("could not add gadget scheme: %w", err)
	// }

	// gadgetClient, err := runtimeclient.New(collector.kubeconfig, runtimeclient.Options{
	// 	Scheme: gadgetScheme,
	// })
	// if err != nil {
	// 	return fmt.Errorf("could not create rest client for gadgets: %w", err)
	// }

	// // Create a gadget.
	// //TODO gadget name should be enum
	// traceName := collector.getTraceName(gadgetName)
	// trace := &gadgetv1alpha1.Trace{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Namespace: "gadget",
	// 		Annotations: map[string]string{
	// 			GadgetOperation: string(gadgetv1alpha1.OperationStart),
	// 		},
	// 		Name: traceName,
	// 	},
	// 	Spec: gadgetv1alpha1.TraceSpec{
	// 		Node:       collector.runtimeInfo.HostNodeName,
	// 		Gadget:     gadgetName,
	// 		RunMode:    gadgetv1alpha1.RunModeManual,
	// 		OutputMode: gadgetv1alpha1.TraceOutputModeStream,
	// 	},
	// }
	// err = gadgetClient.Create(context.TODO(), trace)

	// if err != nil {
	// 	return fmt.Errorf("could not create trace %s: %w", traceName, err)
	// }

	// //TODO watch the trace until it is started
	// //collect output
	// err = collector.runTraceCommandOnPod(gadgetName, gadgetClient, trace)
	// if err != nil {
	// 	log.Printf("\t could not run trace : %s ", err)
	// 	return err
	// }

	// return nil
}
