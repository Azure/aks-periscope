package collector

import (
	"encoding/json"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestSystemPerfCollectorGetName(t *testing.T) {
	const expectedName = "systemperf"

	c := NewSystemPerfCollector(nil, nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("Unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestSystemPerfCollectorCheckSupported(t *testing.T) {
	tests := []struct {
		name          string
		collectorList []string
		wantErr       bool
	}{
		{
			name:          "'connectedCluster' in COLLECTOR_LIST",
			collectorList: []string{"connectedCluster"},
			wantErr:       true,
		},
		{
			name:          "'connectedCluster' not in COLLECTOR_LIST",
			collectorList: []string{},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		runtimeInfo := &utils.RuntimeInfo{
			CollectorList: tt.collectorList,
		}
		c := NewSystemPerfCollector(nil, runtimeInfo)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestSystemPerfCollectorCollect(t *testing.T) {
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

	fixture, _ := test.GetClusterFixture()

	runtimeInfo := &utils.RuntimeInfo{
		CollectorList: []string{},
	}

	c := NewSystemPerfCollector(fixture.ClientConfig, runtimeInfo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()
			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}

			raw := c.GetData()["nodes"]
			var nodeMetrices []NodeMetrics

			if err := json.Unmarshal([]byte(raw), &nodeMetrices); err != nil {
				t.Errorf("unmarshal GetData(): %v", err)
			}

			if len(nodeMetrices) < tt.want {
				t.Errorf("len(GetData()) = %v, want %v", len(nodeMetrices), tt.want)
			}
		})
	}
}
