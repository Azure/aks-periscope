package collector

import (
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestKubeletCmdCollectorGetName(t *testing.T) {
	const expectedName = "kubeletcmd"

	c := NewKubeletCmdCollector(nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("Unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestKubeletCmdCollectorCheckSupported(t *testing.T) {
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
		c := NewKubeletCmdCollector(runtimeInfo)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckSupported() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

func TestKubeletCmdCollectorCollect(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "get kubeletcmd logs",
			want:    1,
			wantErr: true,
		},
	}

	runtimeInfo := &utils.RuntimeInfo{
		OSIdentifier: "linux",
	}
	c := NewKubeletCmdCollector(runtimeInfo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()
			if (err != nil) == tt.wantErr {
				t.Logf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
