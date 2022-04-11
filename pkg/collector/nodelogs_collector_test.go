package collector

import (
	"testing"

	"github.com/Azure/aks-periscope/pkg/utils"
)

func TestNodeLogsCollectorGetName(t *testing.T) {
	const expectedName = "nodelogs"

	c := NewNodeLogsCollector(nil, nil)
	actualName := c.GetName()
	if actualName != expectedName {
		t.Errorf("Unexpected name: expected %s, found %s", expectedName, actualName)
	}
}

func TestNodeLogsCollectorCheckSupported(t *testing.T) {
	c := NewNodeLogsCollector(nil, nil)
	err := c.CheckSupported()
	if err != nil {
		t.Errorf("Error checking supported: %v", err)
	}
}

func TestNodeLogsCollectorCollect(t *testing.T) {
	const (
		file1Name        = "/var/log/test1.log"
		file1ExpectedKey = "var_log_test1.log"
		file1Content     = "Test 1 Content"

		file2Name        = "/var/log/test2.log"
		file2ExpectedKey = "var_log_test2.log"
		file2Content     = "Test 2 Content"
	)

	testLogFiles := map[string]string{
		file1Name: file1Content,
		file2Name: file2Content,
	}

	tests := []struct {
		name      string
		filePaths []string
		wantData  map[string]string
		wantErr   bool
	}{
		{
			name:      "missing first log file",
			filePaths: []string{"/var/log/missing.log", file2Name},
			wantData:  nil,
			wantErr:   true,
		},
		{
			name:      "missing second log file",
			filePaths: []string{file1Name, "/var/log/missing.log"},
			wantData:  nil,
			wantErr:   true,
		},
		{
			name:      "all log files exist",
			filePaths: []string{file1Name, file2Name},
			wantData: map[string]string{
				file1ExpectedKey: file1Content,
				file2ExpectedKey: file2Content,
			},
			wantErr: false,
		},
	}

	reader := utils.NewFakeFileContentReader(testLogFiles)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtimeInfo := &utils.RuntimeInfo{
				NodeLogs: []string{file1Name, file2Name},
			}
			c := NewNodeLogsCollector(runtimeInfo, reader)
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
						t.Errorf("Missing key %s", key)
					}

					if actualValue != expectedValue {
						t.Errorf("Unexpected value for key %s.\nExpected '%s'\nFound '%s'", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}
