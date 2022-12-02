package collector

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
	"k8s.io/client-go/rest"
)

func TestInspektorGadgetDNSTraceCollectorGetName(t *testing.T) {
	const expectedName = "inspektorgadget-dns"

	c := NewInspektorGadgetDNSTraceCollector("", nil, nil, 0)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestInspektorGadgetDNSTraceCollectorCheckSupported(t *testing.T) {
	fixture, _ := test.GetClusterFixture()

	// TODO: test absence of "traces.gadget.kinvolk.io" CRD (maybe by injecting expected CRD name into collector)
	tests := []struct {
		osIdentifier utils.OSIdentifier
		wantErr      bool
	}{
		{
			osIdentifier: utils.Windows,
			wantErr:      true,
		},
		{
			osIdentifier: utils.Linux,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		c := NewInspektorGadgetDNSTraceCollector(tt.osIdentifier, fixture.PeriscopeAccess.ClientConfig, nil, 0)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckSupported() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

func TestInspektorGadgetDNSTraceCollectorCollect(t *testing.T) {
	fixture, _ := test.GetClusterFixture()

	nodeNames, err := fixture.GetNodeNames()
	if err != nil {
		t.Fatalf("Error getting node names: %v", err)
	}

	tests := []struct {
		name           string
		config         *rest.Config
		hostNodeName   string
		setupResources []string
		wantErr        bool
		wantData       map[string]*regexp.Regexp
	}{
		{
			name:           "bad kubeconfig",
			config:         &rest.Config{Host: string([]byte{0})},
			hostNodeName:   "",
			setupResources: []string{},
			wantErr:        true,
			wantData:       nil,
		},
		{
			name:           "valid config",
			config:         fixture.PeriscopeAccess.ClientConfig,
			hostNodeName:   nodeNames[0],
			setupResources: []string{},
			wantErr:        false,
			wantData: map[string]*regexp.Regexp{
				"gadget-dns": regexp.MustCompile(fmt.Sprintf(`^\s*{\s*"node":\s*"%s"`, nodeNames[0])),
			},
		},
		{
			name:         "egress denied",
			config:       fixture.PeriscopeAccess.ClientConfig,
			hostNodeName: nodeNames[0],
			setupResources: []string{
				"/resources/chaos/block-dns-nwp.yaml",
			},
			wantErr: false,
			wantData: map[string]*regexp.Regexp{
				"gadget-dns": regexp.MustCompile(fmt.Sprintf(`^\s*{\s*"node":\s*"%s"`, nodeNames[0])),
			},
		},
	}

	setupResources := func(command string, resources []string) {
		for _, resourcePath := range resources {
			installResourceCommand := fmt.Sprintf("kubectl %s -f %s", command, resourcePath)
			_, err = fixture.CommandRunner.Run(installResourceCommand, fixture.AdminAccess.GetKubeConfigBinding())
			if err != nil {
				t.Fatalf("Error installing resource %s: %v", resourcePath, err)
			}
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupResources("create", tt.setupResources)
			defer setupResources("delete", tt.setupResources)

			runtimeInfo := &utils.RuntimeInfo{
				HostNodeName: tt.hostNodeName,
			}

			c := NewInspektorGadgetDNSTraceCollector(utils.Linux, tt.config, runtimeInfo, time.Second)
			err := c.Collect()

			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
			}
			data := c.GetData()
			test.CompareCollectorData(t, tt.wantData, data)
		})
	}
}
