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
		c := NewIPTablesCollector(runtimeInfo)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckSupported() error = %v, wantErr %v", err, tt.wantErr)
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
		OSIdentifier: "linux",
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
