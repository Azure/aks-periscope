package collector

import (
	"os"
	"path"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

func TestNewSmiCollector(t *testing.T) {
	tests := []struct {
		name          string
		want          int
		wantErr       bool
		collectorName string
	}{
		{
			name:          "get smi crd logs",
			want:          1,
			wantErr:       false,
			collectorName: "smi",
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

	c := NewSmiCollector(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()

			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}

			name := c.GetName()
			if name != tt.collectorName {
				t.Errorf("GetName()) = %v, want %v", name, tt.collectorName)
			}
		})
	}
}
