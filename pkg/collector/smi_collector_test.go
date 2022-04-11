package collector

import (
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestSmiCollectorGetName(t *testing.T) {
	const expectedName = "smi"

	c := NewSmiCollector(nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("Unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestSmiCollectorCheckSupported(t *testing.T) {
	tests := []struct {
		name         string
		osIdentifier string
		collectors   []string
		wantErr      bool
	}{
		{
			name:         "windows",
			osIdentifier: "windows",
			collectors:   []string{"SMI"},
			wantErr:      true,
		},
		{
			name:         "linux without OSM or SMI included",
			osIdentifier: "linux",
			collectors:   []string{"NOT_OSM", "NOT_SMI"},
			wantErr:      true,
		},
		{
			name:         "linux with OSM included",
			osIdentifier: "linux",
			collectors:   []string{"OSM", "NOT_SMI"},
			wantErr:      false,
		},
		{
			name:         "linux with SMI included",
			osIdentifier: "linux",
			collectors:   []string{"NOT_OSM", "SMI"},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		runtimeInfo := &utils.RuntimeInfo{
			OSIdentifier:  tt.osIdentifier,
			CollectorList: tt.collectors,
		}
		c := NewSmiCollector(runtimeInfo)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckSupported() for %s error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestSmiCollectorCollect(t *testing.T) {
	tests := []struct {
		name        string
		want        int
		wantErr     bool
		deployments []*appsv1.Deployment
	}{
		{
			name:        "no SMI deployments found",
			want:        0,
			wantErr:     true,
			deployments: []*appsv1.Deployment{},
		},
	}

	runtimeInfo := &utils.RuntimeInfo{
		OSIdentifier:  "linux",
		CollectorList: []string{"SMI"},
	}

	c := NewSmiCollector(runtimeInfo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := make([]runtime.Object, len(tt.deployments))
			for i := range tt.deployments {
				objs[i] = tt.deployments[i]
			}
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
