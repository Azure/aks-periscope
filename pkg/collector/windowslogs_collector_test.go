package collector

import (
	"fmt"
	"testing"
	"time"

	"github.com/Azure/aks-periscope/pkg/test"
	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestWindowsLogsCollectorGetName(t *testing.T) {
	const expectedName = "windowslogs"

	c := NewWindowsLogsCollector(nil, nil, nil, 0, 0)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestWindowsLogsCollectorCheckSupported(t *testing.T) {
	tests := []struct {
		name         string
		runId        string
		features     []utils.Feature
		osIdentifier string
		wantErr      bool
	}{
		{
			name:         "Run ID not set",
			runId:        "",
			features:     []utils.Feature{utils.WindowsHpc},
			osIdentifier: "windows",
			wantErr:      true,
		},
		{
			name:         "Feature not set",
			runId:        "this_run",
			features:     []utils.Feature{},
			osIdentifier: "windows",
			wantErr:      true,
		},
		{
			name:         "Linux",
			runId:        "this_run",
			features:     []utils.Feature{utils.WindowsHpc},
			osIdentifier: "linux",
			wantErr:      true,
		},
		{
			name:         "Supported",
			runId:        "this_run",
			features:     []utils.Feature{utils.WindowsHpc},
			osIdentifier: "windows",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeInfo := &utils.RuntimeInfo{
				OSIdentifier: tt.osIdentifier,
			}
			c := NewWindowsLogsCollector(runtimeInfo, nil, nil, 0, 0)
			err := c.CheckSupported()
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckSupported() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWindowsLogsCollectorCollect(t *testing.T) {
	const expectedLogOutput = "log file content"
	const runId = "test_run"

	tests := []struct {
		name          string
		timeout       time.Duration
		timeToExport  time.Duration
		exportedFiles map[string]string
		wantErr       bool
		wantData      map[string]string
	}{
		{
			name:         "timeout elapses",
			timeout:      0,
			timeToExport: time.Minute,
			exportedFiles: map[string]string{
				fmt.Sprintf("/output/%s", runId): "",
				"/output/logs/test.log":          expectedLogOutput,
			},
			wantErr:  true,
			wantData: nil,
		},
		{
			name:         "missing logs directory",
			timeout:      time.Minute,
			timeToExport: 0,
			exportedFiles: map[string]string{
				fmt.Sprintf("/output/%s", runId): "",
				"/output/not-in-logs.log":        expectedLogOutput,
			},
			wantErr:  true,
			wantData: nil,
		},
		{
			name:         "successful log collection",
			timeout:      time.Minute,
			timeToExport: 0,
			exportedFiles: map[string]string{
				fmt.Sprintf("/output/%s", runId): "",
				"/output/logs/test.log":          expectedLogOutput,
			},
			wantErr: false,
			wantData: map[string]string{
				"test.log": expectedLogOutput,
			},
		},
	}

	runtimeInfo := &utils.RuntimeInfo{
		RunId:        runId,
		OSIdentifier: "windows",
		Features:     map[utils.Feature]bool{utils.WindowsHpc: true},
	}

	filePaths := &utils.KnownFilePaths{WindowsLogsOutput: "/output"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := test.NewFakeFileContentReader(map[string]string{})

			c := NewWindowsLogsCollector(runtimeInfo, filePaths, reader, time.Microsecond, tt.timeout)
			err := c.Collect()

			if err != nil {
				if !tt.wantErr {
					t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				dataItems := c.GetData()
				for key, expectedValue := range tt.wantData {
					actualValue, ok := dataItems[key]
					if !ok {
						t.Errorf("missing key %s", key)
					}

					if actualValue != expectedValue {
						t.Errorf("unexpected value for key %s.\nExpected '%s'\nFound '%s'", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}
