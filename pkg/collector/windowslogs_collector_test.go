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

	c := NewWindowsLogsCollector("", nil, nil, nil, 0, 0)
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
		osIdentifier utils.OSIdentifier
		wantErr      bool
	}{
		{
			name:         "Run ID not set",
			runId:        "",
			features:     []utils.Feature{utils.WindowsHpc},
			osIdentifier: utils.Windows,
			wantErr:      true,
		},
		{
			name:         "Feature not set",
			runId:        "this_run",
			features:     []utils.Feature{},
			osIdentifier: utils.Windows,
			wantErr:      true,
		},
		{
			name:         "Linux",
			runId:        "this_run",
			features:     []utils.Feature{utils.WindowsHpc},
			osIdentifier: utils.Linux,
			wantErr:      true,
		},
		{
			name:         "Supported",
			runId:        "this_run",
			features:     []utils.Feature{utils.WindowsHpc},
			osIdentifier: utils.Windows,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeInfo := &utils.RuntimeInfo{
				RunId:    tt.runId,
				Features: map[utils.Feature]bool{},
			}
			for _, feature := range tt.features {
				runtimeInfo.Features[feature] = true
			}

			c := NewWindowsLogsCollector(tt.osIdentifier, runtimeInfo, nil, nil, 0, 0)
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
	notificationPath := fmt.Sprintf("/output/%s", runId)

	tests := []struct {
		name          string
		exportedFiles map[string]string
		errorPaths    []string
		wantErr       bool
		wantData      map[string]string
	}{
		{
			name: "timeout elapses - no completion notification",
			exportedFiles: map[string]string{
				"/output/logs/test.log": expectedLogOutput,
			},
			errorPaths: []string{},
			wantErr:    true,
			wantData:   nil,
		},
		{
			name: "missing logs directory",
			exportedFiles: map[string]string{
				notificationPath:          "",
				"/output/not-in-logs.log": expectedLogOutput,
			},
			errorPaths: []string{},
			wantErr:    true,
			wantData:   nil,
		},
		{
			name: "list files error",
			exportedFiles: map[string]string{
				notificationPath:        "",
				"/output/logs/test.log": expectedLogOutput,
			},
			errorPaths: []string{"/output/logs"},
			wantErr:    true,
			wantData:   nil,
		},
		{
			name: "read log files error",
			exportedFiles: map[string]string{
				notificationPath:        "",
				"/output/logs/test.log": expectedLogOutput,
			},
			errorPaths: []string{"/output/logs/test.log"},
			wantErr:    true,
			wantData:   nil,
		},
		{
			name: "successful log collection",
			exportedFiles: map[string]string{
				notificationPath:        "",
				"/output/logs/test.log": expectedLogOutput,
			},
			errorPaths: []string{},
			wantErr:    false,
			wantData: map[string]string{
				"collect-windows-logs/test.log": expectedLogOutput,
			},
		},
	}

	runtimeInfo := &utils.RuntimeInfo{
		RunId:    runId,
		Features: map[utils.Feature]bool{utils.WindowsHpc: true},
	}

	filePaths := &utils.KnownFilePaths{WindowsLogsOutput: "/output"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := test.NewFakeFileSystem(map[string]string{})

			c := NewWindowsLogsCollector(utils.Windows, runtimeInfo, filePaths, fs, time.Microsecond, time.Second)

			for path, content := range tt.exportedFiles {
				fs.AddOrUpdateFile(path, content)
			}

			for _, path := range tt.errorPaths {
				fs.SetFileAccessError(path, fmt.Errorf("expected error accessing %s", path))
			}

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
						continue
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
