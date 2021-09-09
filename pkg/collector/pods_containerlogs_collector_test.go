package collector

import (
	"os"
	"testing"
)

func TestPodsContainerLogsCollector(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "get pods container logs",
			want:    1,
			wantErr: false,
		},
	}

	config, err := getConfig()
	if err != nil {
		t.Errorf("cannot get kube config: %w", err)
	}

	c := NewPodsContainerLogs(config)

	if err := os.Setenv("DIAGNOSTIC_CONTAINERLOGS_LIST", "kube-system"); err != nil {
		t.Fatalf("Setenv: %v", err)
	}

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
