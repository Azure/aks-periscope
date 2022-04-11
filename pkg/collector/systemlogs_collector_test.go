package collector

import (
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestSystemLogsCollectorGetName(t *testing.T) {
	const expectedName = "systemlogs"

	c := NewSystemLogsCollector(nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("Unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestSystemLogsCollectorCheckSupported(t *testing.T) {
	tests := []struct {
		osIdentifier string
		wantErr      bool
	}{
		{
			osIdentifier: "windows",
			wantErr:      true,
		},
		{
			osIdentifier: "linux",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		runtimeInfo := &utils.RuntimeInfo{
			OSIdentifier: tt.osIdentifier,
		}
		c := NewSystemLogsCollector(runtimeInfo)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckSupported() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

func TestSystemLogsCollectorCollect(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "get system logs",
			want:    1,
			wantErr: true,
		},
	}

	runtimeInfo := &utils.RuntimeInfo{
		OSIdentifier: "linux",
	}

	c := NewSystemLogsCollector(runtimeInfo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()
			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
