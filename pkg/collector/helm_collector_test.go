package collector

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

func TestHelmCollectorGetName(t *testing.T) {
	const expectedName = "helm"

	c := NewHelmCollector(nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("Unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestHelmCollectorCheckSupported(t *testing.T) {
	c := NewHelmCollector(nil)
	err := c.CheckSupported()
	if err != nil {
		t.Errorf("Error checking supported: %v", err)
	}
}

func TestHelmCollectorCollect(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "get release history",
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

	c := NewHelmCollector(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()
			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}

			raw := c.GetData()["helm_list"]
			var releases []HelmRelease

			if err := json.Unmarshal([]byte(raw), &releases); err != nil {
				t.Errorf("unmarshal GetData(): %v", err)
			}

			if len(releases) < tt.want {
				t.Errorf("len(GetData()) = %v, want %v", len(releases), tt.want)
			}
		})
	}
}
