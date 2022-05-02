package collector

import (
	"testing"
)

func TestNetworkOutboundCollectorGetName(t *testing.T) {
	const expectedName = "networkoutbound"

	c := NewNetworkOutboundCollector()
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestNetworkOutboundCollectorCheckSupported(t *testing.T) {
	c := NewNetworkOutboundCollector()
	err := c.CheckSupported()
	if err != nil {
		t.Errorf("error checking supported: %v", err)
	}
}

func TestNetworkOutboundCollectorCollect(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "get networkbound logs",
			want:    1,
			wantErr: false,
		},
	}

	c := NewNetworkOutboundCollector()

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
