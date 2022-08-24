package collector

import (
	"testing"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestDNSCollectorGetName(t *testing.T) {
	const expectedName = "dns"

	c := NewDNSCollector("", nil, nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestDNSCollectorCheckSupported(t *testing.T) {
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
		c := NewDNSCollector(tt.osIdentifier, nil, nil)
		err := c.CheckSupported()
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckSupported() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

func TestDNSCollectorCollect(t *testing.T) {
	const expectedHostConfContent = "hostconf"
	const expectedContainerConfContent = "containerconf"

	tests := []struct {
		name     string
		files    map[string]string
		wantErr  bool
		wantData map[string]string
	}{
		{
			name: "missing host conf",
			files: map[string]string{
				"/etc/resolv.conf": expectedContainerConfContent,
			},
			wantErr:  true,
			wantData: nil,
		},
		{
			name: "missing container conf",
			files: map[string]string{
				"/host/etc/resolv.conf": expectedHostConfContent,
			},
			wantErr:  true,
			wantData: nil,
		},
		{
			name: "existing files",
			files: map[string]string{
				"/host/etc/resolv.conf": expectedHostConfContent,
				"/etc/resolv.conf":      expectedContainerConfContent,
			},
			wantErr: false,
			wantData: map[string]string{
				"virtualmachine": expectedHostConfContent,
				"kubernetes":     expectedContainerConfContent,
			},
		},
	}

	filePaths := &utils.KnownFilePaths{
		ResolvConfHost:      "/host/etc/resolv.conf",
		ResolvConfContainer: "/etc/resolv.conf",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := test.NewFakeFileSystem(tt.files)

			c := NewDNSCollector(utils.Linux, filePaths, fs)
			err := c.Collect()

			if err != nil {
				if !tt.wantErr {
					t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				dataItems := c.GetData()
				for key, expectedValue := range tt.wantData {
					result, ok := dataItems[key]
					if !ok {
						t.Errorf("missing key %s", key)
					}

					testDataValue(t, result, func(actualValue string) {
						if actualValue != expectedValue {
							t.Errorf("unexpected value for key %s.\nExpected '%s'\nFound '%s'", key, expectedValue, actualValue)
						}
					})
				}
			}
		})
	}
}
