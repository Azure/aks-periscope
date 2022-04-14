package collector

import (
	"os"
	"path"
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
	"k8s.io/client-go/tools/clientcmd"
)

func TestKubeObjectsCollectorGetName(t *testing.T) {
	const expectedName = "kubeobjects"

	c := NewKubeObjectsCollector(nil, nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("Unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestKubeObjectsCollectorCheckSupported(t *testing.T) {
	c := NewKubeObjectsCollector(nil, nil)
	err := c.CheckSupported()
	if err != nil {
		t.Errorf("Error checking supported: %v", err)
	}
}

func TestKubeObjectsCollectorCollect(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "get kube objects logs",
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

	runtimeInfo := &utils.RuntimeInfo{
		KubernetesObjects: []string{"kube-system/pod", "kube-system/service", "kube-system/deployment"},
	}

	c := NewKubeObjectsCollector(config, runtimeInfo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()

			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}
			raw := c.GetData()

			if len(raw) < tt.want {
				t.Errorf("len(GetData()) = %v, want %v", len(raw), tt.want)
			}
		})
	}
}
