package action

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// ConnectivityType defines a network connectivity test object
type ConnectivityType struct {
	Type string `json:"Type"`
	URL  string `json:"URL"`
}

// ConnectivityData defines a data object for network connectivity info
type ConnectivityData struct {
	TimeStamp time.Time `json:"TimeStamp"`
	ConnectivityType
	Connected bool   `json:"Connected"`
	Error     string `json:"Error"`
}

// CollectIntervalInSeconds defines interval for collect metrics
var CollectIntervalInSeconds = 5

// ConnectivityDiagnosticData defines network connectivity diagnostic datum
type ConnectivityDiagnosticData struct {
	Type  string    `json:"Type"`
	Start time.Time `json:"Start"`
	End   time.Time `json:"End"`
	Error string    `json:"Error"`
}

// ProcessIntervalInSeconds defines interval for process network connectivity metrics
var ProcessIntervalInSeconds = 30

// NetworkConnectivityAction defines an action on container logs
type NetworkConnectivityAction struct{}

var _ interfaces.Action = &NetworkConnectivityAction{}

// GetName implements the interface method
func (action *NetworkConnectivityAction) GetName() string {
	return "networkconnectivity"
}

// Collect implements the interface method
func (action *NetworkConnectivityAction) Collect() ([]string, error) {
	APIServerFQDN, err := utils.GetAPIServerFQDN()
	if err != nil {
		return nil, err
	}

	connectivityTypes := []ConnectivityType{}
	connectivityTypes = append(connectivityTypes,
		ConnectivityType{
			Type: "OutboundConnectivity",
			URL:  "google.com:80",
		},
	)
	connectivityTypes = append(connectivityTypes,
		ConnectivityType{
			Type: "APIServerConnectivity",
			URL:  "kubernetes.default.svc.cluster.local:443",
		},
	)
	connectivityTypes = append(connectivityTypes,
		ConnectivityType{
			Type: "TunnelConnectivity",
			URL:  APIServerFQDN + ":9000",
		},
	)
	connectivityTypes = append(connectivityTypes,
		ConnectivityType{
			Type: "ACRConnectivity",
			URL:  "azurecr.io:80",
		},
	)
	connectivityTypes = append(connectivityTypes,
		ConnectivityType{
			Type: "MCRConnectivity",
			URL:  "mcr.microsoft.com:80",
		},
	)

	networkConnectivityFiles := make([]string, 0)

	rootPath, _ := utils.CreateCollectorDir(action.GetName())

	for _, connectivityType := range connectivityTypes {
		networkConnectivityFile := filepath.Join(rootPath, connectivityType.Type)

		go func(connectivityType ConnectivityType, file string) {
			ticker := time.NewTicker(time.Duration(CollectIntervalInSeconds) * time.Second)
			f, _ := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			defer f.Close()

			for {
				select {
				case <-ticker.C:
					tcpDial(connectivityType, f)
				}
			}
		}(connectivityType, networkConnectivityFile)

		networkConnectivityFiles = append(networkConnectivityFiles, networkConnectivityFile)
	}

	return networkConnectivityFiles, nil
}

func tcpDial(connectivityType ConnectivityType, f *os.File) {
	timeout := time.Duration(5 * time.Second)
	_, err := net.DialTimeout("tcp", connectivityType.URL, timeout)

	// only write when connection failed
	if err != nil {
		data := &ConnectivityData{
			TimeStamp:        time.Now().Truncate(1 * time.Second),
			ConnectivityType: connectivityType,
			Connected:        err == nil,
			Error:            err.Error(),
		}

		dataBytes, _ := json.Marshal(data)
		f.WriteString(string(dataBytes) + "\n")
	}
}

// Process implements the interface method
func (action *NetworkConnectivityAction) Process(files []string) error {
	rootPath, _ := utils.CreateDiagnosticDir()
	networkConnectivityDiagnosticFile := filepath.Join(rootPath, action.GetName())

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
			var connectivityData ConnectivityData
			json.Unmarshal([]byte(scanner.Text()), &connectivityData)

			if dataPoint.Start.IsZero() {
				setDataPoint(&connectivityData, &dataPoint)
			} else {
				if connectivityData.Error != dataPoint.Error {
					connectivityDiagnosticData = append(connectivityDiagnosticData, dataPoint)
					setDataPoint(&connectivityData, &dataPoint)
				} else {
					if int(connectivityData.TimeStamp.Sub(dataPoint.End).Seconds()) > CollectIntervalInSeconds {
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

func setDataPoint(connectivityData *ConnectivityData, dataPoint *ConnectivityDiagnosticData) {
	dataPoint.Type = connectivityData.Type
	dataPoint.Start = connectivityData.TimeStamp
	dataPoint.End = connectivityData.TimeStamp
	dataPoint.Error = connectivityData.Error
}
