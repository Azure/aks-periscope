package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Azure/aks-periscope/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// PodsContainerLogsCollector defines a Pods Container Logs Collector struct
type PodsContainerLogsCollector struct {
	kubeconfig *restclient.Config
	data       map[string]string
}

type PodsContainerStruct struct {
	Name          string        `json:"name"`
	Ready         string        `json:"ready"`
	Status        string        `json:"status"`
	Restart       int32         `json:"restart"`
	Age           time.Duration `json:"age"`
	ContainerName string        `json:"containerName"`
	ContainerLog  string        `json:"containerLog"`
}

// NewPodsContainerLogs is a constructor
func NewPodsContainerLogs(config *restclient.Config) *PodsContainerLogsCollector {
	return &PodsContainerLogsCollector{
		data:       make(map[string]string),
		kubeconfig: config,
	}
}

func (collector *PodsContainerLogsCollector) GetName() string {
	return "podscontainerlogs"
}

// Collect implements the interface method
func (collector *PodsContainerLogsCollector) Collect() error {
	containerNamespaces := strings.Fields(os.Getenv("DIAGNOSTIC_CONTAINERLOGS_LIST"))

	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(collector.kubeconfig)
	if err != nil {
		return fmt.Errorf("getting access to K8S failed: %w", err)
	}

	for _, namespace := range containerNamespaces {
		// List the pods in the given namespace
		podList, err := utils.GetPods(clientset, namespace)

		if err != nil {
			return fmt.Errorf("getting pods failed: %w", err)
		}

		// List all the pods similar to kubectl get pods -n <my namespace>
		for _, pod := range podList.Items {
			// Calculate the age of the pod
			podCreationTime := pod.GetCreationTimestamp()
			age := time.Since(podCreationTime.Time).Round(time.Second)

			// Get the status of each of the pods
			podStatus := pod.Status

			var containerRestarts int32
			var containerReady int

			// If a pod has multiple containers, get the status from all
			for container := range pod.Spec.Containers {
				containerRestarts += podStatus.ContainerStatuses[container].RestartCount

				if podStatus.ContainerStatuses[container].Ready {
					containerReady++
				}
			}
			containerName := pod.Spec.Containers[0].Name
			// Get pods container logs
			containerLogs, err := getPodContainerLogs(namespace, pod.Name, containerName, clientset)

			if err != nil {
				return fmt.Errorf("getting container logs failed: %w", err)
			}

			podsContainerData := &PodsContainerStruct{
				Name:          pod.Name,
				Ready:         fmt.Sprintf("%v/%v", containerReady, len(pod.Spec.Containers)),
				Status:        string(podStatus.Phase),
				Restart:       containerRestarts,
				Age:           age,
				ContainerName: containerName,
				ContainerLog:  containerLogs,
			}

			data, err := json.Marshal(podsContainerData)
			if err != nil {
				return fmt.Errorf("marshalling podsContainerData: %w", err)
			}

			// Append this to data to be printed in a table
			collector.data[pod.Name] = string(data)
		}
	}

	return nil
}

func (collector *PodsContainerLogsCollector) GetData() map[string]string {
	return collector.data
}

func getPodContainerLogs(
	namespace string,
	podName string,
	containerName string,
	clientset *kubernetes.Clientset) (string, error) {

	count := int64(100)
	podLogOptions := v1.PodLogOptions{
		Container: containerName,
		TailLines: &count,
	}

	podLogRequest := clientset.CoreV1().
		Pods(namespace).
		GetLogs(podName, &podLogOptions)
	stream, err := podLogRequest.Stream(context.Background())

	if err != nil {
		return "", fmt.Errorf("getting pod logs request failed: %w", err)
	}
	defer stream.Close()
	returnData := ""
	for {
		buf := make([]byte, 2000)
		numBytes, err := stream.Read(buf)

		if err == io.EOF {
			break
		}

		if err != nil {
			return "", fmt.Errorf("pod logs stream read failure: %w", err)
		}
		returnData = string(buf[:numBytes])
	}

	return returnData, err
}
