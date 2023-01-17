package collector

import (
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
	containercollection "github.com/inspektor-gadget/inspektor-gadget/pkg/container-collection"
	ocispec "github.com/opencontainers/runtime-spec/specs-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func TestInspektorGadgetDNSTraceCollectorGetName(t *testing.T) {
	const expectedName = "inspektorgadget-dns"

	c := NewInspektorGadgetDNSTraceCollector("", nil, nil, []containercollection.ContainerCollectionOption{})
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestInspektorGadgetDNSTraceCollectorCheckSupported(t *testing.T) {
	tests := []struct {
		osIdentifier utils.OSIdentifier
		wantErr      bool
	}{
		{
			osIdentifier: utils.Windows,
			wantErr:      true,
		},
		{
			osIdentifier: utils.Linux,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		c := NewInspektorGadgetDNSTraceCollector(tt.osIdentifier, nil, nil, []containercollection.ContainerCollectionOption{})
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckSupported() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

func TestInspektorGadgetDNSTraceCollectorCollect(t *testing.T) {
	fixture, _ := test.GetClusterFixture()

	nodeNames, err := fixture.GetNodeNames()
	if err != nil {
		t.Fatalf("Error getting node names: %v", err)
	}

	nodeName := nodeNames[0]

	// Ensure there is a pod running on our node. This pod process won't actually be traced, but we'll pretend
	// our DNS queries are coming from this pod.
	setupCommands := []string{
		fmt.Sprintf("kubectl -n %s apply -f /resources/pause-daemonset.yaml", fixture.KnownNamespaces.Periscope),
		fmt.Sprintf("kubectl -n %s rollout status daemonset pauseds --timeout=60s", fixture.KnownNamespaces.Periscope),
	}
	setupCommand := strings.Join(setupCommands, " && ")
	_, err = fixture.CommandRunner.Run(setupCommand, fixture.AdminAccess.GetKubeConfigBinding())
	if err != nil {
		t.Fatalf("Error installing test daemonset: %v", err)
	}

	// Find the pod we've created.
	pod, err := getTestPod(fixture, nodeName)
	if err != nil {
		t.Fatalf("Unable to get test pod: %v", err)
	}

	domains := []string{"microsoft.com", "google.com", "shouldnotexist.com"}

	tests := []struct {
		name         string
		hostNodeName string
		wantErr      bool
		wantData     map[string]*regexp.Regexp
	}{
		{
			name:         "valid",
			hostNodeName: nodeName,
			wantErr:      false,
			wantData: map[string]*regexp.Regexp{
				"dnstracer": getExpectedDnsTraceData(fixture, nodeName, pod, domains),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeInfo := &utils.RuntimeInfo{
				HostNodeName: tt.hostNodeName,
			}

			// Use a channel to get hold of the ContainerCollection (via a ContainerCollectionOption).
			// We'll use this to add our pretend kubernetes container to.
			ccChan := make(chan *containercollection.ContainerCollection)

			// Create a container whose PID is the current process, but has the attributes of the pod.
			testContainer := getCurrentProcessAsKubernetesContainer(pod)

			// Make use of container collection options to:
			// - Get hold of the ContainerCollection
			// - Add node name to the collection
			// - Add K8s pod metadata to the containers (based on the pod UID)
			// We skip the following options that Periscope uses:
			// - WithRuncFanotify: because we're not assuming our test is running in a containerized environment.
			// - WithInitialKubernetesContainers: because this uses an in-cluster context that is inaccessible here.
			// - WithPodInformer: also uses an in-cluster context.
			// - WithCgroupEnrichment: because this process is not expected to have cgroups, and we're faking those.
			// - WithLinuxNamespaceEnrichment: because this will set HostNetwork to true, and we're pretending it's false.
			opts := []containercollection.ContainerCollectionOption{
				withContainerCollectionReceiver(ccChan),
				containercollection.WithNodeName(tt.hostNodeName),
				containercollection.WithKubernetesEnrichment(tt.hostNodeName, fixture.PeriscopeAccess.ClientConfig),
			}

			waiter := func() {
				// While the tracer is running, add a fake container and perform some DNS queries from its process.
				cc := <-ccChan
				cc.AddContainer(testContainer)
				defer cc.RemoveContainer(testContainer.ID)

				for _, domain := range domains {
					// Perform a DNS lookup (discarding the result because we're only testing the events it triggers)
					net.LookupIP(domain)
				}
			}

			c := NewInspektorGadgetDNSTraceCollector(utils.Linux, runtimeInfo, waiter, opts)
			err := c.Collect()

			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}
			data := c.GetData()
			compareCollectorData(t, tt.wantData, data)
		})
	}
}

func getExpectedDnsTraceData(fixture *test.ClusterFixture, nodeName string, pod *corev1.Pod, domains []string) *regexp.Regexp {
	containerName := pod.Spec.Containers[0].Name

	eventPatterns := []string{}
	eventPatterns = append(eventPatterns,
		fmt.Sprintf(
			`{"node":%q,"namespace":%q,"pod":%q,"container":%q,"type":"debug","message":"tracer attached"}`,
			nodeName, fixture.KnownNamespaces.Periscope, pod.Name, containerName),
	)

	for _, domain := range domains {
		eventPatterns = append(eventPatterns,
			fmt.Sprintf(
				`{"node":%q,"namespace":%q,"pod":%q,"container":%q,"type":"normal","id":"[\d\w\.]+","qr":"Q","nameserver":"[\d\.]+","pktType":"OUTGOING","qtype":"A","name":%q}`,
				nodeName, fixture.KnownNamespaces.Periscope, pod.Name, containerName, domain+"."),
		)
	}

	pattern := strings.Join(eventPatterns, `(\n.*)*`)
	return regexp.MustCompile(pattern)
}

func getTestPod(fixture *test.ClusterFixture, nodeName string) (*corev1.Pod, error) {
	fieldSelector := fields.OneTermEqualSelector("spec.nodeName", nodeName)
	nameRequirement, err := labels.NewRequirement("name", selection.Equals, []string{"pauseds"})
	if err != nil {
		return nil, fmt.Errorf("failed to create name requirement: %w", err)
	}
	labelSelector := labels.NewSelector().Add(*nameRequirement)
	pods, err := fixture.AdminAccess.Clientset.CoreV1().Pods(fixture.KnownNamespaces.Periscope).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve test pod: %w", err)
	}
	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("expected one test pod, found %d", len(pods.Items))
	}
	return &pods.Items[0], nil
}

func getCurrentProcessAsKubernetesContainer(pod *corev1.Pod) *containercollection.Container {
	// Pretend the current test process is a kubernetes container.
	testProcessPid := os.Getpid()
	return &containercollection.Container{
		ID: fmt.Sprintf("test%08d", testProcessPid),
		// Namespace, Podname and Labels should be populated by the Kubernetes enricher.
		Pid: uint32(testProcessPid),
		// This would normally be added by the cgroup enricher
		CgroupV2: fmt.Sprintf("/kubelet/kubepods/pod%s/k8scontainerid", pod.ObjectMeta.UID),
		// If OciConfig.Mounts is populated, it's used by the Kubernetes enricher to set the container name
		// when the container is added.
		// In gadgettracermanager, whose behaviour we're trying to replicate, the RuncFanotify enricher
		// populates this. If we don't populate it here, the kubernetes enricher 'drops' the container:
		// (https://github.com/inspektor-gadget/inspektor-gadget/blob/08b450065bb839e33012d80d476b3a3c17946379/pkg/container-collection/options.go#L497-L500)
		OciConfig: &ocispec.Spec{
			Mounts: []ocispec.Mount{
				ocispec.Mount{
					Destination: "/dev/termination-log",
					Type:        "bind",
					Source:      fmt.Sprintf("/var/lib/kubelet/pods/%s/containers/%s/dnstest/a1234abcd", pod.ObjectMeta.UID, pod.Spec.Containers[0].Name),
					Options:     []string{"rbind", "rprivate", "rw"},
				},
			},
		},
	}
}

func withContainerCollectionReceiver(ccChan chan *containercollection.ContainerCollection) containercollection.ContainerCollectionOption {
	return func(cc *containercollection.ContainerCollection) error {
		go func() {
			ccChan <- cc
		}()
		return nil
	}
}
