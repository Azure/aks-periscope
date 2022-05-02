package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/aks-periscope/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// PodsContainerLogsCollector defines a Pods Container Logs Collector struct
type PodsContainerLogsCollector struct {
	data        map[string]string
	kubeconfig  *restclient.Config
	runtimeInfo *utils.RuntimeInfo
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
func NewPodsContainerLogsCollector(config *restclient.Config, runtimeInfo *utils.RuntimeInfo) *PodsContainerLogsCollector {
	return &PodsContainerLogsCollector{
		data:        make(map[string]string),
		kubeconfig:  config,
		runtimeInfo: runtimeInfo,
	}
}

func (collector *PodsContainerLogsCollector) GetName() string {
	return "podscontainerlogs"
}

func (collector *PodsContainerLogsCollector) CheckSupported() error {
	if !utils.Contains(collector.runtimeInfo.CollectorList, "connectedCluster") {
		return fmt.Errorf("not included because 'connectedCluster' not in COLLECTOR_LIST variable. Included values: %s", strings.Join(collector.runtimeInfo.CollectorList, " "))
	}

	return nil
}

// Collect implements the interface method
func (collector *PodsContainerLogsCollector) Collect() error {
	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(collector.kubeconfig)
	if err != nil {
		return fmt.Errorf("getting access to K8S failed: %w", err)
	}

	for _, namespace := range collector.runtimeInfo.ContainerLogsNamespaces {
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
			for _, containerItem := range pod.Spec.Containers {
				containerName := containerItem.Name
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
				collector.data[pod.Name+"-"+containerName] = string(data)
			}
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

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)

	if err != nil {
		return "", fmt.Errorf("pod logs stream read failure: %w", err)
	}

	returnData = buf.String()

	return returnData, err
}
