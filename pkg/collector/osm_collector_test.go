package collector

import (
	"os"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestOsmCollector(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
		deployments []*appsv1.Deployment
		collectorName string
	}{
		{
			name:    "no deployments found",
			want:    0,
			wantErr: true,
			deployments: []*appsv1.Deployment{},
			collectorName: "osm",
		},
	}

	c := NewOsmCollector()

	if err := os.Setenv("COLLECTOR_LIST", "OSM"); err != nil {
		t.Fatalf("Setenv: %v", err)
	}

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
			name := c.GetName()
			if name != tt.collectorName {
				t.Errorf("GetName()) = %v, want %v", name, tt.collectorName)
			}
		})
	}
}
