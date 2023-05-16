package collector

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/aks-periscope/pkg/test"
	containercollection "github.com/inspektor-gadget/inspektor-gadget/pkg/container-collection"
	containerutils "github.com/inspektor-gadget/inspektor-gadget/pkg/container-utils"
	ocispec "github.com/opencontainers/runtime-spec/specs-go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/rest"
)

func getTestContainerCollectionOptions(
	ccChan chan *containercollection.ContainerCollection,
	nodeName string,
	clusterConfig *rest.Config,
) []containercollection.ContainerCollectionOption {
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
	return []containercollection.ContainerCollectionOption{
		withContainerCollectionReceiver(ccChan),
		containercollection.WithNodeName(nodeName),
		containercollection.WithKubernetesEnrichment(nodeName, clusterConfig),
	}
}

func ensurePod(fixture *test.ClusterFixture, nodeName string) (*corev1.Pod, error) {
	setupCommands := []string{
		fmt.Sprintf("kubectl -n %s apply -f /resources/pause-daemonset.yaml", fixture.KnownNamespaces.Periscope),
		fmt.Sprintf("kubectl -n %s rollout status daemonset pauseds --timeout=60s", fixture.KnownNamespaces.Periscope),
	}
	setupCommand := strings.Join(setupCommands, " && ")
	_, err := fixture.CommandRunner.Run(setupCommand, fixture.AdminAccess.GetKubeConfigBinding())
	if err != nil {
		return nil, fmt.Errorf("failed to install test daemonset: %w", err)
	}

	// Find the pod we've created.
	return getTestPod(fixture, nodeName)
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

func getCurrentProcessAsKubernetesContainer(pod *corev1.Pod) (*containercollection.Container, error) {
	// Pretend the current test process is a kubernetes container.
	testProcessPid := os.Getpid()
	mnsnsid, err := containerutils.GetMntNs(testProcessPid)
	if err != nil {
		return nil, fmt.Errorf("failed to get mnt namespace id: %w", err)
	}

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
				{
					Destination: "/dev/termination-log",
					Type:        "bind",
					Source:      fmt.Sprintf("/var/lib/kubelet/pods/%s/containers/%s/dnstest/a1234abcd", pod.ObjectMeta.UID, pod.Spec.Containers[0].Name),
					Options:     []string{"rbind", "rprivate", "rw"},
				},
			},
		},
		// Set to correlate this container with requests coming from the TCP tracer.
		Mntns: mnsnsid,
	}, nil
}

func withContainerCollectionReceiver(ccChan chan *containercollection.ContainerCollection) containercollection.ContainerCollectionOption {
	return func(cc *containercollection.ContainerCollection) error {
		go func() {
			ccChan <- cc
		}()
		return nil
	}
}
