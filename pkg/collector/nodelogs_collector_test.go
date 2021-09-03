package collector

import (
	"os"
	"testing"
)

func TestNodeLogsCollector(t *testing.T) {
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

	c := NewNodeLogsCollector()

	if err := os.Setenv("DIAGNOSTIC_NODELOGS_LIST", "/var/log/cloud-init.log"); err != nil {
		t.Fatalf("Setenv: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := os.Stat("/var/lib/kubelet/kubeconfig"); os.IsExist(err) {
				err := c.Collect()

				if (err != nil) != tt.wantErr {
					t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
				}
				raw := c.GetData()

				if len(raw) < tt.want {
					t.Errorf("len(GetData()) = %v, want %v", len(raw), tt.want)
				}
			}
		})
	}
}
