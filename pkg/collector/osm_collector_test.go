package collector

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
)

func TestOsmCollectorGetName(t *testing.T) {
	const expectedName = "osm"

	c := NewOsmCollector(nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestOsmCollectorCheckSupported(t *testing.T) {
	tests := []struct {
		name         string
		osIdentifier string
		collectors   []string
		wantErr      bool
	}{
		{
			name:         "windows",
			osIdentifier: "windows",
			collectors:   []string{"OSM"},
			wantErr:      true,
		},
		{
			name:         "linux without OSM included",
			osIdentifier: "linux",
			collectors:   []string{"NOT_OSM"},
			wantErr:      true,
		},
		{
			name:         "linux with OSM included",
			osIdentifier: "linux",
			collectors:   []string{"OSM"},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		runtimeInfo := &utils.RuntimeInfo{
			OSIdentifier:  tt.osIdentifier,
			CollectorList: tt.collectors,
		}
		c := NewOsmCollector(runtimeInfo)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckSupported() for %s error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func setupOsmTest(t *testing.T) *test.ClusterFixture {
	fixture, _ := test.GetClusterFixture()

	// Run commands to wait until the OSM application rollout is complete
	commands := []string{
		fmt.Sprintf("kubectl rollout status -n %s deploy/bookbuyer --timeout=240s", fixture.KnownNamespaces.OsmBookBuyer),
		fmt.Sprintf("kubectl rollout status -n %s deploy/bookthief --timeout=240s", fixture.KnownNamespaces.OsmBookThief),
		fmt.Sprintf("kubectl rollout status -n %s deploy/bookstore --timeout=240s", fixture.KnownNamespaces.OsmBookStore),
		fmt.Sprintf("kubectl rollout status -n %s deploy/bookstore-v2 --timeout=240s", fixture.KnownNamespaces.OsmBookStore),
		fmt.Sprintf("kubectl rollout status -n %s deploy/bookwarehouse --timeout=240s", fixture.KnownNamespaces.OsmBookWarehouse),
		fmt.Sprintf("kubectl rollout status -n %s statefulset/mysql --timeout=240s", fixture.KnownNamespaces.OsmBookWarehouse),
	}

	_, err := fixture.CommandRunner.Run(strings.Join(commands, " && "), fixture.GetKubeConfigBinding())
	if err != nil {
		t.Fatalf("Error waiting for OSM application rollout to complete: %v", err)
	}

	return fixture
}

func TestOsmCollectorCollect(t *testing.T) {
	tests := []struct {
		name        string
		want        int
		wantErr     bool
		deployments []*appsv1.Deployment
	}{
		{
			name:    "OSM deployments found",
			want:    107,
			wantErr: false,
		},
	}

	fixture := setupOsmTest(t)
	os.Setenv("KUBECONFIG", fixture.KubeConfigFile.Name())

	runtimeInfo := &utils.RuntimeInfo{
		OSIdentifier:  "linux",
		CollectorList: []string{"OSM"},
	}

	c := NewOsmCollector(runtimeInfo)

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
