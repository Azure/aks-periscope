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
	"k8s.io/client-go/rest"
)

func TestInspektorGadgetDNSTraceCollectorGetName(t *testing.T) {
	const expectedName = "inspektorgadget-dns"

	c := NewInspektorGadgetDNSTraceCollector("", nil, nil, nil, []containercollection.ContainerCollectionOption{})
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestInspektorGadgetDNSTraceCollectorCheckSupported(t *testing.T) {
	fixture, _ := test.GetClusterFixture()

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
		c := NewInspektorGadgetDNSTraceCollector(tt.osIdentifier, fixture.PeriscopeAccess.ClientConfig, nil, nil, []containercollection.ContainerCollectionOption{})
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

	tests := []struct {
		name         string
		config       *rest.Config
		hostNodeName string
		wantErr      bool
		wantData     map[string]*regexp.Regexp
	}{
		{
			name:         "bad kubeconfig",
			config:       &rest.Config{Host: string([]byte{0})},
			hostNodeName: "",
			wantErr:      true,
			wantData:     nil,
		},
		{
			name:         "valid config",
			config:       fixture.PeriscopeAccess.ClientConfig,
			hostNodeName: nodeName,
			wantErr:      false,
			wantData:     map[string]*regexp.Regexp{},
		},
	}

	setupCommands := []string{
		fmt.Sprintf("kubectl -n %s apply -f /resources/pause-daemonset.yaml", fixture.KnownNamespaces.Periscope),
		fmt.Sprintf("kubectl -n %s rollout status daemonset pauseds --timeout=60s", fixture.KnownNamespaces.Periscope),
	}
	setupCommand := strings.Join(setupCommands, " && ")
	_, err = fixture.CommandRunner.Run(setupCommand, fixture.AdminAccess.GetKubeConfigBinding())
	if err != nil {
		t.Fatalf("Error installing test daemonset: %v", err)
	}

	pod, err := getTestPod(fixture, nodeName)
	if err != nil {
		t.Fatalf("Unable to get test pod: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeInfo := &utils.RuntimeInfo{
				HostNodeName: tt.hostNodeName,
			}

			testsready := make(chan struct{})
			testsdone := make(chan struct{})
			waiter := func() {
				close(testsready)
				<-testsdone
			}

			option := getDNSTestContainerOption(pod, testsready, testsdone)
			c := NewInspektorGadgetDNSTraceCollector(utils.Linux, tt.config, runtimeInfo, waiter, []containercollection.ContainerCollectionOption{option})
			err := c.Collect()

			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}
			data := c.GetData()
			compareCollectorData(t, tt.wantData, data)
		})
	}
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

func getDNSTestContainerOption(pod *corev1.Pod, ready chan struct{}, done chan struct{}) func(*containercollection.ContainerCollection) error {
	domains := []string{"microsoft.com", "google.com", "shouldnotexist.com"}

	return func(cc *containercollection.ContainerCollection) error {
		go func() {
			// Wait until the Collect function has set up its trace and is ready to listen for events
			<-ready

			// Pretend the test process is a container which has just been launched.
			testProcessPid := os.Getpid()
			testContainer := containercollection.Container{
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

			// Adding the container will invoke the other ContainerCollectionOptions, which will populate the k8s container properties.
			cc.AddContainer(&testContainer)

			// HostNetwork will have been set to true by now (because the 'container' and host PIDs are the same).
			// For testing purposes we want it to be false, because the event callback checks this when populating
			// the event properties.
			testContainer.HostNetwork = false

			for _, domain := range domains {
				// Perform a DNS lookup (discarding the result because we're only testing the events it triggers)
				net.LookupIP(domain)
			}

			cc.RemoveContainer(testContainer.ID)
			close(done)
		}()

		return nil
	}
}
