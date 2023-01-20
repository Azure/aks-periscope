package collector

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
	containercollection "github.com/inspektor-gadget/inspektor-gadget/pkg/container-collection"
	corev1 "k8s.io/api/core/v1"
)

func TestInspektorGadgetTCPTraceCollectorGetName(t *testing.T) {
	const expectedName = "inspektorgadget-tcptrace"

	c := NewInspektorGadgetTCPTraceCollector("", nil, nil, []containercollection.ContainerCollectionOption{})
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestInspektorGadgetTCPTraceCollectorCheckSupported(t *testing.T) {
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
		c := NewInspektorGadgetTCPTraceCollector(tt.osIdentifier, nil, nil, []containercollection.ContainerCollectionOption{})
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckSupported() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

func TestInspektorGadgetTCPTraceCollectorCollect(t *testing.T) {
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
				"tcptracer": getExpectedTcpTraceData(fixture, nodeName, pod, testContainer),
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

				// Make an HTTP request to a non-localhost URL - this should produce a TCP connect event.
				url := "https://mcr.microsoft.com/v2/aks/periscope/tags/list"
				_, err = http.Get(url)
				if err != nil {
					t.Fatalf("Unable to make request to %s: %v", url, err)
				}
			}

			c := NewInspektorGadgetTCPTraceCollector(utils.Linux, runtimeInfo, waiter, opts)
			err := c.Collect()

			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}
			data := c.GetData()
			compareCollectorData(t, tt.wantData, data)
		})
	}
}

func getExpectedTcpTraceData(fixture *test.ClusterFixture, nodeName string, pod *corev1.Pod, container *containercollection.Container) *regexp.Regexp {
	containerName := pod.Spec.Containers[0].Name

	pattern := fmt.Sprintf(
		`{"node":%q,"namespace":%q,"pod":%q,"container":%q,"type":"normal","operation":"connect","pid":\d+,"comm":"[\w.]+","ipversion":4,"saddr":"[\d\.]+","daddr":"[\d\.]+","sport":\d+,"dport":443,"mountnsid":%d}`,
		nodeName, fixture.KnownNamespaces.Periscope, pod.Name, containerName, container.Mntns)

	return regexp.MustCompile(pattern)
}
