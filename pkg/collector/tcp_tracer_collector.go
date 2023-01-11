package collector

import (
	"fmt"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"github.com/cilium/ebpf/rlimit"
	containercollection "github.com/inspektor-gadget/inspektor-gadget/pkg/container-collection"
	containerutils "github.com/inspektor-gadget/inspektor-gadget/pkg/container-utils"
	runtimeclient "github.com/inspektor-gadget/inspektor-gadget/pkg/container-utils/runtime-client"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/tcp/types"
	tracercollection "github.com/inspektor-gadget/inspektor-gadget/pkg/tracer-collection"
	restclient "k8s.io/client-go/rest"
	"runtime"
)

const traceName = "trace_exec"

type TCPConnectionInfo struct {
	connectionType string `json:"connectionType"`
	process        string `json:"process"`
	ipVersion      string `json:"ip"`
	source         string `json:"source"`
	destination    string `json:"destination"`
}

// TCPTracerCollector defines a TCP tracer Collector struct
type TCPTracerCollector struct {
	data        map[string]string
	kubeconfig  *restclient.Config
	runtimeInfo *utils.RuntimeInfo
}

// NewTCPTracerCollector is a constructor to collect TCP trace data using IG
func NewTCPTracerCollector(config *restclient.Config, runtimeInfo *utils.RuntimeInfo) *TCPTracerCollector {
	return &TCPTracerCollector{
		data:        make(map[string]string),
		kubeconfig:  config,
		runtimeInfo: runtimeInfo,
	}
}

func (collector *TCPTracerCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}

func (collector *TCPTracerCollector) GetName() string {
	return "tcptracer"
}

func (collector *TCPTracerCollector) CheckSupported() error {
	// check for OS since ebpf is only available on linux
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return fmt.Errorf("ebpf is not yet supported on windows or macos")
	}
	if err := rlimit.RemoveMemlock(); err != nil {
		return err
	}

	return nil
}

// Collect implements the interface method
func (collector *TCPTracerCollector) Collect() error {
	//
	//eventCallback := func(event types.Event) {
	//	fmt.Printf("A new TCP connection event IPv%d between source %s:%d -> destination %s:%d. Caused by command %s(%d) of type %s\n ",
	//		event.IPVersion, event.Saddr, event.Sport, event.Daddr, event.Dport, event.Comm, event.Pid, event.Operation)
	//}
	//
	//tracer, err := tracer.NewTracer()
	//if err != nil {
	//	return fmt.Errorf("error creating tracer: %v", err)
	//}
	//defer tracer.Stop()
	//
	//clientset, err := kubernetes.NewForConfig(collector.kubeconfig)
	//if err != nil {
	//	return fmt.Errorf("getting access to K8S failed: %w", err)
	//}
	//
	//ctxBackground := context.Background()
	//
	//namespacesList, err := clientset.CoreV1().Namespaces().List(ctxBackground, metav1.ListOptions{})
	//if err != nil {
	//	return fmt.Errorf("unable to list namespaces in the cluster: %w", err)
	//}
	//
	//for _, namespace := range namespacesList.Items {
	//
	//	pods, err := clientset.CoreV1().Pods(namespace.Name).List(ctxBackground, metav1.ListOptions{})
	//
	//	if err != nil {
	//		return fmt.Errorf("listing Pods error: %w", err)
	//	}
	//
	//	for _, i := range pods.Items {
	//		for _, container := range i.Spec.Containers {
	//			containerName := container.Name
	//			setTracerContext(eventCallback, containerName)
	//		}
	//		pdbresult := make([]PDBInfo, 0)
	//		pdbinfo := PDBInfo{
	//			Name:               i.Name,
	//			MinAvailable:       i.Spec.MinAvailable.String(),
	//			MaxUnavailable:     i.Spec.MaxUnavailable.String(),
	//			DisruptionsAllowed: i.Status.DisruptionsAllowed,
	//		}
	//		pdbresult = append(pdbresult, pdbinfo)
	//	}
	//
	//	if err != nil {
	//		return fmt.Errorf("marshall PDB to json: %w", err)
	//	}
	//	collector.data[fmt.Sprintf("tcp-%s-%s-%s", namespace.Name, podName, containerName)] = string(data)
	//}
	//
	return nil
}

func setTracerContext(eventCallback func(event types.Event), containerName string) error {

	// Create and initialize the container collection
	containers := &containercollection.ContainerCollection{}

	tracerCollection, err := tracercollection.NewTracerCollection(containers)
	if err != nil {
		fmt.Printf("failed to create trace-collection: %s\n", err)
		return err
	}
	defer tracerCollection.Close()

	// Define the different options for the container collection instance
	opts := []containercollection.ContainerCollectionOption{
		// Indicate the callback that will be invoked each time there is an event
		containercollection.WithPubSub(tracerCollection.TracerMapsUpdater()),
		containercollection.WithRuncFanotify(),

		// Enrich events with Linux namespaces information
		containercollection.WithLinuxNamespaceEnrichment(),
		containercollection.WithMultipleContainerRuntimesEnrichment(
			[]*containerutils.RuntimeConfig{
				{Name: runtimeclient.DockerName},
				{Name: runtimeclient.ContainerdName},
			}),
	}

	if err := containers.Initialize(opts...); err != nil {
		fmt.Printf("failed to initialize container collection: %s\n", err)
		return err
	}
	defer containers.Close()

	err = setTracerByContainer(containerName, containers, eventCallback, tracerCollection)
	return err
}

func setTracerByContainer(containerName string, containerCollection *containercollection.ContainerCollection, eventCallback func(event types.Event),
	tracerCollection *tracercollection.TracerCollection) error {
	//
	//// Create a tracer instance.
	//containerSelector := containercollection.ContainerSelector{
	//	Name: containerName,
	//}
	//
	//if err := tracerCollection.AddTracer(traceName, containerSelector); err != nil {
	//	fmt.Printf("error adding tracer: %s\n", err)
	//	return err
	//}
	//defer tracerCollection.RemoveTracer(traceName)
	//
	//// Get mount namespace map to filter by containers
	//mountnsmap, err := tracerCollection.TracerMountNsMap(traceName)
	//if err != nil {
	//	fmt.Printf("failed to get mountnsmap: %s\n", err)
	//	return err
	//}
	//
	//// Create the tracer
	//tracer, err := tracer.NewTracer(&tracer.Config{MountnsMap: mountnsmap}, containerSelector, eventCallback)
	//if err != nil {
	//	fmt.Printf("error creating tracer: %s\n", err)
	//	return err
	//}
	//defer tracer.Stop()
	//
	//// Graceful shutdown
	//exit := make(chan os.Signal, 1)
	//signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	//<-exit
	return nil
}
