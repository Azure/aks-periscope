package processor

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/collector"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// ConnectivityDiagnosticData defines network connectivity diagnostic data
type ConnectivityDiagnosticData struct {
	Type  string    `json:"Type"`
	Start time.Time `json:"Start"`
	End   time.Time `json:"End"`
	Error string    `json:"Error"`
}

// ProcessIntervalInSeconds defines interval for process network connectivity metrics
var ProcessIntervalInSeconds = 30

// ProcessNetworkConnectivity processes network connectivity metrics
func ProcessNetworkConnectivity(name string, files []string) error {
	rootPath, _ := utils.CreateDiagnosticDir()
	networkConnectivityDiagnosticFile := filepath.Join(rootPath, name)

	ticker := time.NewTicker(time.Duration(ProcessIntervalInSeconds) * time.Second)
	for {
		select {
		case <-ticker.C:
			processNetworkConnectivity(files, networkConnectivityDiagnosticFile)
		}
	}
}

func processNetworkConnectivity(files []string, networkConnectivityDiagnosticFile string) error {
	f, _ := os.OpenFile(networkConnectivityDiagnosticFile, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()

	var connectivityDiagnosticData []ConnectivityDiagnosticData

	for _, file := range files {
		t, _ := os.Open(file)
		defer t.Close()

		dataPoint := ConnectivityDiagnosticData{}
		scanner := bufio.NewScanner(t)
		for scanner.Scan() {
			var connectivityData collector.ConnectivityData
			json.Unmarshal([]byte(scanner.Text()), &connectivityData)

			if dataPoint.Start.IsZero() {
				setDataPoint(&connectivityData, &dataPoint)
			} else {
				if connectivityData.Error != dataPoint.Error {
					connectivityDiagnosticData = append(connectivityDiagnosticData, dataPoint)
					setDataPoint(&connectivityData, &dataPoint)
				} else {
					if int(connectivityData.TimeStamp.Sub(dataPoint.End).Seconds()) > collector.CollectIntervalInSeconds {
						connectivityDiagnosticData = append(connectivityDiagnosticData, dataPoint)
						setDataPoint(&connectivityData, &dataPoint)
					} else {
						dataPoint.End = connectivityData.TimeStamp
					}
				}
			}
		}

		if !dataPoint.Start.IsZero() {
			connectivityDiagnosticData = append(connectivityDiagnosticData, dataPoint)
		}
	}

	for _, dataPoint := range connectivityDiagnosticData {
		dataBytes, _ := json.Marshal(dataPoint)
		f.WriteString(string(dataBytes) + "\n")
	}

	return nil
}

func setDataPoint(connectivityData *collector.ConnectivityData, dataPoint *ConnectivityDiagnosticData) {
	dataPoint.Type = connectivityData.Type
	dataPoint.Start = connectivityData.TimeStamp
	dataPoint.End = connectivityData.TimeStamp
	dataPoint.Error = connectivityData.Error
}
