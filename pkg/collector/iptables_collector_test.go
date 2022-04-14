package collector

import (
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestIPTablesCollectorGetName(t *testing.T) {
	const expectedName = "iptables"

	c := NewIPTablesCollector(nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("Unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestIPTablesCollectorCheckSupported(t *testing.T) {
	tests := []struct {
		name          string
		osIdentifier  string
		collectorList []string
		wantErr       bool
	}{
		{
			name:          "windows",
			osIdentifier:  "windows",
			collectorList: []string{"connectedCluster"},
			wantErr:       true,
		},
		{
			name:          "'connectedCluster' in COLLECTOR_LIST",
			osIdentifier:  "linux",
			collectorList: []string{"connectedCluster"},
			wantErr:       true,
		},
		{
			name:          "'connectedCluster' not in COLLECTOR_LIST",
			osIdentifier:  "linux",
			collectorList: []string{},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		runtimeInfo := &utils.RuntimeInfo{
			OSIdentifier:  tt.osIdentifier,
			CollectorList: tt.collectorList,
		}
		c := NewIPTablesCollector(runtimeInfo)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestIPTablesCollectorCollect(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "get iptables logs",
			want:    1,
			wantErr: true,
		},
	}

	runtimeInfo := &utils.RuntimeInfo{
		OSIdentifier:  "linux",
		CollectorList: []string{},
	}
	c := NewIPTablesCollector(runtimeInfo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()
			if (err != nil) == tt.wantErr {
				t.Logf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
