package collector

import (
	"testing"
)

func TestNewSystemLogsCollector(t *testing.T) {
	tests := []struct {
		name          string
		want          int
		wantErr       bool
		collectorName string
	}{
		{
			name:          "get system logs",
			want:          1,
			wantErr:       false,
			collectorName: "systemlogs",
		},
	}

	c := NewSystemLogsCollector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()
			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Get Data test needs to be written as the current test dont run at node level.

			name := c.GetName()
			if name != tt.collectorName {
				t.Errorf("GetName()) = %v, want %v", name, tt.collectorName)
			}
		})
	}
}
