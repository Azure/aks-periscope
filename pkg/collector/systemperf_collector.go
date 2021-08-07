package collector

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

// SystemPerfCollector defines a SystemPerf Collector struct
type SystemPerfCollector struct {
	kubeconfig *restclient.Config
	data       map[string]string
}

type NodeMetrics struct {
	NodeName    string `json:"name"`
	CPUUsage    int64  `json:"cpuusage"`
	MemoryUsage int64  `json:"memoryusage"`
}

type PodMetrics struct {
	ContainerName string `json:"name"`
	CPUUsage      int64  `json:"cpuusage"`
	MemoryUsage   int64  `json:"memoryusage"`
}

// NewSystemPerfCollector is a constructor
func NewSystemPerfCollector(config *restclient.Config) *SystemPerfCollector {
	return &SystemPerfCollector{
		data:       make(map[string]string),
		kubeconfig: config,
	}
}

func (collector *SystemPerfCollector) GetName() string {
	return "systemperf"
}

// Collect implements the interface method
func (collector *SystemPerfCollector) Collect() error {
	metric, err := metrics.NewForConfig(collector.kubeconfig)
	if err != nil {
		return fmt.Errorf("metrics for config error: %w", err)
	}

	// Node Metrics collector
	nodeMetrics, err := metric.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("node metrics error: %w", err)
	}

	noderesult := make([]NodeMetrics, 0)

	for _, nodeMetric := range nodeMetrics.Items {
		cpuQuantity := nodeMetric.Usage.Cpu().MilliValue()
		memQuantity, ok := nodeMetric.Usage.Memory().AsInt64()
		if !ok {
			return err
		}

		nm := NodeMetrics{
			NodeName:    nodeMetric.Name,
			CPUUsage:    cpuQuantity,
			MemoryUsage: memQuantity,
		}

		noderesult = append(noderesult, nm)
	}
	jsonNodeResult, err := json.Marshal(noderesult)
	if err != nil {
		return fmt.Errorf("marshall node metrics to json: %w", err)
	}

	collector.data["nodes"] = string(jsonNodeResult)

	// Pod Metrics collector
	podMetrics, err := metric.MetricsV1beta1().PodMetricses(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("pod metrics failure: %w", err)
	}

	podresult := make([]PodMetrics, 0)

	for _, podMetric := range podMetrics.Items {
		podContainers := podMetric.Containers
		for _, container := range podContainers {
			cpuQuantity := container.Usage.Cpu().MilliValue()
			memQuantity, ok := container.Usage.Memory().AsInt64()
			if !ok {
				return fmt.Errorf("usage memory failure: %w", err)
			}

			pm := PodMetrics{
				ContainerName: container.Name,
				CPUUsage:      cpuQuantity,
				MemoryUsage:   memQuantity,
			}

			podresult = append(podresult, pm)
		}
	}
	jsonPodResult, err := json.Marshal(podresult)
	if err != nil {
		return fmt.Errorf("marshall pod metrics to json: %w", err)
	}

	collector.data["pods"] = string(jsonPodResult)

	return nil
}

func (collector *SystemPerfCollector) GetData() map[string]string {
	return collector.data
}
