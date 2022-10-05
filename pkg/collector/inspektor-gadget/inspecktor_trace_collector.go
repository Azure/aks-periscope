package inspektor_gadget

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
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

func init() {
	rand.Seed(time.Now().UnixNano())
}

// InspektorGadgetTraceCollector defines a InspektorGadget Trace Collector that are common to trace gadgets
type InspektorGadgetTraceCollector struct {
	data          map[string]string
	kubeconfig    *restclient.Config
	commandRunner *utils.KubeCommandRunner
	runtimeInfo   *utils.RuntimeInfo
}

func (collector *InspektorGadgetTraceCollector) runTraceCommandOnPod(traceGadgetName string,
	gadgetClient runtimeclient.Client,
	trace *gadgetv1alpha1.Trace,
	collectingPeriod time.Duration) error {

	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(collector.kubeconfig)
	if err != nil {
		return fmt.Errorf("getting access to K8S failed: %w", err)
	}

	gadgetPods, err := clientset.CoreV1().Pods("gadget").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list gadget pods: %w", err)
	}
	var command = []string{"./bin/gadgettracermanager", "-call", "receive-stream", "-tracerid", fmt.Sprintf("trace_gadget_%s", traceGadgetName)}

	collectorGrp := new(sync.WaitGroup)

	for _, pod := range gadgetPods.Items {

		collectorGrp.Add(1)
		go func(podName string) {
			defer collectorGrp.Done()

			stdout := new(bytes.Buffer)
			stderr := new(bytes.Buffer)
			streamOptions := remotecommand.StreamOptions{
				Stdout: stdout,
				Stderr: stderr,
			}

			request := clientset.CoreV1().RESTClient().Post().
				Resource("pods").
				Name(podName).
				Namespace("gadget").
				SubResource("exec").
				VersionedParams(&v1.PodExecOptions{
					Stdin:   false,
					Stdout:  true,
					Stderr:  true,
					TTY:     false,
					Command: command,
				}, scheme.ParameterCodec)

			log.Printf("\tPost request to trace stream : %s ", request.URL())
			exec, err := remotecommand.NewSPDYExecutor(collector.kubeconfig, "POST", request.URL())

			if err != nil {
				log.Printf("\tError creating SPDYExecutor for pod exec %q: %v", podName, err)
				return
			}

			err = exec.Stream(streamOptions)
			if err != nil {
				log.Printf("\tObtain trace stream erred: %s, %v. Try a different pod ", podName, err)
				return
			}

			log.Printf("\tCollecting trace stream %s from pod %s", traceGadgetName, podName)
			result := strings.TrimSpace(stdout.String()) + "\n" + strings.TrimSpace(stderr.String())
			result = strings.TrimSpace(result)
			collector.data[fmt.Sprintf("%s-%s", traceGadgetName, podName)] = result
			log.Printf("\tCollected trace stream %s from pod %s", traceGadgetName, podName)
		}(pod.Name)
	}

	//TODO kill in a proper way by apply annotation
	log.Printf("\twait for %v to stop collection", collectingPeriod)
	time.Sleep(collectingPeriod)

	err = gadgetClient.Delete(context.TODO(), trace)
	if err != nil {
		log.Printf("could not kill trace %s: %v", trace.Name, err)
	}

	// wait for the final result to be written
	collectorGrp.Wait()

	return nil
}

func (collector *InspektorGadgetTraceCollector) randomPodID() string {
	output := make([]byte, 5)
	allowedCharacters := "0123456789abcdefghijklmnopqrstuvwxyz"
	for i := range output {
		output[i] = allowedCharacters[rand.Int31n(int32(len(allowedCharacters)))]
	}
	return string(output)
}

func (collector *InspektorGadgetTraceCollector) CheckSupported() error {
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

func (collector *InspektorGadgetTraceCollector) collect(gadgetName string, collectionPeriod time.Duration) error {

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

	// Create a gadget.
	//TODO gadget name should be enum
	traceGadgetName := fmt.Sprintf("%s-%s", gadgetName, collector.randomPodID())
	trace := &gadgetv1alpha1.Trace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "gadget",
			Annotations: map[string]string{
				GadgetOperation: string(gadgetv1alpha1.OperationStart),
			},
			Name: traceGadgetName,
		},
		Spec: gadgetv1alpha1.TraceSpec{
			Node:       collector.runtimeInfo.HostNodeName,
			Gadget:     gadgetName,
			RunMode:    gadgetv1alpha1.RunModeManual,
			OutputMode: gadgetv1alpha1.TraceOutputModeStream,
		},
	}
	err = gadgetClient.Create(context.TODO(), trace)

	if err != nil {
		return fmt.Errorf("could not create trace %s: %w", traceGadgetName, err)
	}

	//TODO watch the trace until it is started
	//collect output
	err = collector.runTraceCommandOnPod(traceGadgetName, gadgetClient, trace, collectionPeriod)
	if err != nil {
		log.Printf("\t could not run trace : %s ", err)
		return err
	}

	return nil
}
