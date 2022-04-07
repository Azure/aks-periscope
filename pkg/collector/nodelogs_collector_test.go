package collector

import (
	"fmt"
	"os"
	"testing"
)

func TestNodeLogsCollector(t *testing.T) {
	// TODO: Avoid using real files for testing. These are chosen because they happen to exist in
	// common Linux distros (including Ubuntu on WSL) as well as the Ubuntu GitHub workflow host,
	// but they won't exist in every environment we might wish to run tests.
	file1 := "/var/log/alternatives.log"
	file1Key := "var_log_alternatives.log"

	file2 := "/var/log/dpkg.log"
	file2Key := "var_log_dpkg.log"

	tests := []struct {
		name          string
		wantKeys      []string
		wantErr       bool
		collectorName string
	}{
		{
			name:          "get node logs",
			wantKeys:      []string{file1Key, file2Key},
			wantErr:       false,
			collectorName: "nodelogs",
		},
	}

	c := NewNodeLogsCollector()

	if err := os.Setenv("DIAGNOSTIC_NODELOGS_LIST", fmt.Sprintf("%s %s", file1, file2)); err != nil {
		t.Fatalf("Setenv: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()

			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}

			dataItems := c.GetData()
			if len(dataItems) != len(tt.wantKeys) {
				t.Errorf("len(GetData()) = %v, want %v", len(dataItems), len(tt.wantKeys))
			}

			for _, fileKey := range tt.wantKeys {
				_, ok := dataItems[fileKey]
				if !ok {
					t.Errorf("Missing file key %s", fileKey)
				}
			}

			name := c.GetName()
			if name != tt.collectorName {
				t.Errorf("GetName()) = %v, want %v", name, tt.collectorName)
			}
		})
	}
}
