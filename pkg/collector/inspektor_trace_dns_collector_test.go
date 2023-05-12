package collector

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
	containercollection "github.com/inspektor-gadget/inspektor-gadget/pkg/container-collection"
	corev1 "k8s.io/api/core/v1"
)

func TestInspektorGadgetDNSTraceCollectorGetName(t *testing.T) {
	const expectedName = "inspektorgadget-dnstrace"

	c := NewInspektorGadgetDNSTraceCollector(nil, nil, []containercollection.ContainerCollectionOption{})
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestInspektorGadgetDNSTraceCollectorCheckSupported(t *testing.T) {
	c := NewInspektorGadgetDNSTraceCollector(nil, nil, []containercollection.ContainerCollectionOption{})
	err := c.CheckSupported()
	if err != nil {
		t.Errorf("error checking supported: %v", err)
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
	// our traces are coming from this pod.
	pod, err := ensurePod(fixture, nodeName)
	if err != nil {
		t.Fatalf("Error setting up test pod: %v", err)
	}

	// Create a container whose PID is the current process, but has the attributes of the pod.
	testContainer, err := getCurrentProcessAsKubernetesContainer(pod)
	if err != nil {
		t.Fatalf("Unable create container from current process: %v", err)
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

			opts := getTestContainerCollectionOptions(ccChan, tt.hostNodeName, fixture.PeriscopeAccess.ClientConfig)

			waiter := func() {
				// While the tracer is running, add a fake container and perform some DNS queries from its process.
				cc := <-ccChan
				cc.AddContainer(testContainer)
				defer cc.RemoveContainer(testContainer.ID)

				for _, domain := range domains {
					// Perform a DNS lookup (discarding the result because we're only testing the events it triggers)
					_, _ = net.LookupIP(domain)
				}
			}

			c := NewInspektorGadgetDNSTraceCollector(runtimeInfo, waiter, opts)
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
				`{"node":%q,"namespace":%q,"pod":%q,"container":%q,"type":"normal","id":"[\w.]+","qr":"Q","nameserver":"[\d\.]+","pktType":"OUTGOING","qtype":"A","name":%q}`,
				nodeName, fixture.KnownNamespaces.Periscope, pod.Name, containerName, domain+"."),
		)
	}

	pattern := strings.Join(eventPatterns, `(\n.*)*`)
	return regexp.MustCompile(pattern)
}
