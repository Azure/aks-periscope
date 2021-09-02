package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// PODSContainerLogsCollector defines a Pods Container Logs Collector struct
type PODSContainerLogsCollector struct {
	kubeconfig *restclient.Config
	data       map[string]string
}

type PODSContainerStruct struct {
	Name          string
	Ready         string
	Status        string
	Restart       int32
	Age           time.Duration
	ContainerName string
	ContainerLog  string
}

// NewPODSContainerLogs is a constructor
func NewPODSContainerLogs(config *restclient.Config) *PODSContainerLogsCollector {
	return &PODSContainerLogsCollector{
		data:       make(map[string]string),
		kubeconfig: config,
	}
}

func (collector *PODSContainerLogsCollector) GetName() string {
	return "podscontainerlogs"
}

// Collect implements the interface method
func (collector *PODSContainerLogsCollector) Collect() error {
	containerNamespaces := strings.Fields(os.Getenv("DIAGNOSTIC_CONTAINERLOGS_LIST"))

	// Creates the clientset
	clientset, err := kubernetes.NewForConfig(collector.kubeconfig)
	if err != nil {
		return fmt.Errorf("error in getting access to K8S: %v", err)
	}

	for _, namespace := range containerNamespaces {
		// List the pods in the given namespace
		podList, err := getPods(clientset, namespace)

		if err != nil {
			return fmt.Errorf("error while getting pods: %v", err)
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
				return fmt.Errorf("error while getting container logs: %v", err)
			}

			podsContainerData := &PODSContainerStruct{
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
				return fmt.Errorf("error in marshalling podsContainerData: %v", err)
			}

			// Append this to data to be printed in a table
			collector.data[pod.Name] = string(data)
		}
	}

	return nil
}

func (collector *PODSContainerLogsCollector) GetData() map[string]string {
	return collector.data
}

func getPods(clientset *kubernetes.Clientset, namespace string) (*v1.PodList, error) {
	// Create a pod interface for the given namespace
	podInterface := clientset.CoreV1().Pods(namespace)

	// List the pods in the given namespace
	podList, err := podInterface.List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, fmt.Errorf("error in getting pods: %v", err)
	}

	return podList, nil
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
		return "", fmt.Errorf("error in getting pod logs request: %v", err)
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
			return "", fmt.Errorf("error in pod logs stream read: %v", err)
		}
		returnData = string(buf[:numBytes])
	}

	return returnData, err
}
