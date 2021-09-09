package collector

import (
	"encoding/json"
	"os"
	"testing"
)

func TestSystemperfCollector(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "get node logs",
			want:    1,
			wantErr: false,
		},
	}

	config, err := getConfig()
	if err != nil {
		t.Errorf("cannot get kube config: %w", err)
	}

	c := NewSystemPerfCollector(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := os.Stat("/var/lib/kubelet/kubeconfig"); os.IsExist(err) {
				err := c.Collect()
				// This test will not work in kind cluster.
				// For kind cluster use in CI build:
				// message: "metrics error: the server could not find the requested resource (get nodes.metrics.k8s.io)"
				// hence skipping this for CI.

				if (err != nil) != tt.wantErr {
					t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
				}

				raw := c.GetData()["nodes"]
				var nodeMetrices []NodeMetrics

				if err := json.Unmarshal([]byte(raw), &nodeMetrices); err != nil {
					t.Errorf("unmarshal GetData(): %v", err)
				}

				if len(nodeMetrices) < tt.want {
					t.Errorf("len(GetData()) = %v, want %v", len(nodeMetrices), tt.want)
				}
			}
		})
	}
}
