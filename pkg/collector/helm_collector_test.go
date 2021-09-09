package collector

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func TestHelmCollector(t *testing.T) {
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

	config, err := getConfig()
	if err != nil {
		t.Errorf("cannot get kube config: %w", err)
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

func getConfig() (*restclient.Config, error) {
	inCluster := os.Getenv("IN_CLUSTER")

	var config *restclient.Config
	if inCluster == "1" {
		var err error
		config, err = restclient.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		dirname, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		master := ""
		kubeconfig := path.Join(dirname, ".kube/config")
		config, err = clientcmd.BuildConfigFromFlags(master, kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}
