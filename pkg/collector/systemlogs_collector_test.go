package collector

import (
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestSystemLogsCollectorGetName(t *testing.T) {
	const expectedName = "systemlogs"

	c := NewSystemLogsCollector("", nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestSystemLogsCollectorCheckSupported(t *testing.T) {
	tests := []struct {
		name          string
		osIdentifier  utils.OSIdentifier
		collectorList []string
		wantErr       bool
	}{
		{
			name:          "windows",
			osIdentifier:  utils.Windows,
			collectorList: []string{"connectedCluster"},
			wantErr:       true,
		},
		{
			name:          "'connectedCluster' in COLLECTOR_LIST",
			osIdentifier:  utils.Linux,
			collectorList: []string{"connectedCluster"},
			wantErr:       true,
		},
		{
			name:          "'connectedCluster' not in COLLECTOR_LIST",
			osIdentifier:  utils.Linux,
			collectorList: []string{},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		runtimeInfo := &utils.RuntimeInfo{
			CollectorList: tt.collectorList,
		}
		c := NewSystemLogsCollector(tt.osIdentifier, runtimeInfo)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
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
		CollectorList: []string{},
	}

	c := NewSystemLogsCollector(utils.Linux, runtimeInfo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()
			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
