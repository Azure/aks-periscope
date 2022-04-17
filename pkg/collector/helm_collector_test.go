package collector

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
	"k8s.io/client-go/rest"
)

func setup(t *testing.T) (*rest.Config, string, func()) {
	fixture, _ := test.GetClusterFixture()

	chartDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Error creating chart directory: %v", err)
	}

	err = test.CopyDir(test.TestChart, "resources/testchart", chartDir)
	if err != nil {
		t.Fatalf("Error copying testchart files to %s: %v", chartDir, err)
	}

	namespace, err := fixture.CreateNamespace("helmtest")
	if err != nil {
		t.Fatalf("Error creating namespace: %v", err)
	}

	installChartCommand, installHelmBinds := test.GetInstallHelmChartCommand("test", namespace, chartDir, fixture.KubeConfigFile.Name())
	installChartOutput, err := fixture.CommandRunner.Run(installChartCommand, installHelmBinds...)
	if err != nil {
		t.Fatalf("Error installing helm chart: %v", err)
	}

	log.Printf("OUTPUT:\n%s", installChartOutput)

	teardown := func() {
		os.RemoveAll(chartDir)
	}

	return fixture.ClientConfig, chartDir, teardown
}

func TestHelmCollectorGetName(t *testing.T) {
	const expectedName = "helm"

	c := NewHelmCollector(nil, nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("Unexpected name: expected %s, found %s", expectedName, actualName)
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
	clientConfig, _, teardown := setup(t)
	defer teardown()

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
