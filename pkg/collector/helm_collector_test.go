package collector

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
	"k8s.io/client-go/rest"
)

const (
	namespacePrefix = "helmtest"
	releaseName     = "helmtest-release"
)

func setupHelmTest(t *testing.T) *rest.Config {
	fixture, _ := test.GetClusterFixture()

	namespace, err := fixture.CreateTestNamespace(namespacePrefix)
	if err != nil {
		t.Fatalf("Error creating test namespace %s: %v", namespace, err)
	}

	installChartCommand := fmt.Sprintf("helm install %s /resources/testchart --namespace %s", releaseName, namespace)
	_, err = fixture.CommandRunner.Run(installChartCommand, fixture.AdminAccess.GetKubeConfigBinding())
	if err != nil {
		t.Fatalf("Error installing helm release %s into %s namespace: %v", releaseName, namespace, err)
	}

	return fixture.PeriscopeAccess.ClientConfig
}

func TestHelmCollectorGetName(t *testing.T) {
	const expectedName = "helm"

	c := NewHelmCollector(nil, nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestHelmCollectorCheckSupported(t *testing.T) {
	tests := []struct {
		name          string
		collectorList []string
		wantErr       bool
	}{
		{
			name:          "'connectedCluster' in COLLECTOR_LIST",
			collectorList: []string{"connectedCluster"},
			wantErr:       false,
		},
		{
			name:          "'connectedCluster' not in COLLECTOR_LIST",
			collectorList: []string{},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		runtimeInfo := &utils.RuntimeInfo{
			CollectorList: tt.collectorList,
		}
		c := NewHelmCollector(nil, runtimeInfo)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestHelmCollectorCollect(t *testing.T) {
	clientConfig := setupHelmTest(t)

	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{
			name:    "get release history",
			want:    1,
			wantErr: false,
		},
	}

	runtimeInfo := &utils.RuntimeInfo{
		CollectorList: []string{"connectedCluster"},
	}

	c := NewHelmCollector(clientConfig, runtimeInfo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Collect()
			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}

			raw := c.GetData()["helm_list"]
			var releases []HelmRelease

			if err := json.Unmarshal([]byte(raw), &releases); err != nil {
				t.Errorf("unmarshal GetData(): %v", err)
			}

			if len(releases) < tt.want {
				t.Errorf("len(GetData()) = %v, want %v", len(releases), tt.want)
			}
		})
	}
}
