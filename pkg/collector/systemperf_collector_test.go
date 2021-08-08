package collector

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
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

	dirname, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Cannot get user home dir: %v", err)
	}

	master := ""
	kubeconfig := path.Join(dirname, ".kube/config")
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		t.Fatalf("Cannot load kube config: %v", err)
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
