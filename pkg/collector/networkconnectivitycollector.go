package collector

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"time"

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

// CollectNetworkConnectivity collects network connectivity metrics
func CollectNetworkConnectivity(name string) ([]string, error) {
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

	rootPath, _ := utils.CreateCollectorDir(name)

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
