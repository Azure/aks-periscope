package action

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type networkOutboundType struct {
	Type string `json:"Type"`
	URL  string `json:"URL"`
}

type networkOutboundDatum struct {
	TimeStamp time.Time `json:"TimeStamp"`
	networkOutboundType
	Connected bool   `json:"Connected"`
	Error     string `json:"Error"`
}

type networkOutboundDiagnosticDatum struct {
	Type  string    `json:"Type"`
	Start time.Time `json:"Start"`
	End   time.Time `json:"End"`
	Error string    `json:"Error"`
}

type networkOutboundAction struct {
	name                     string
	collectIntervalInSeconds int
	collectCountForProcess   int
	collectCountForExport    int
	exporter                 interfaces.Exporter
	collectFiles             []string
	processFiles             []string
}

var _ interfaces.Action = &networkOutboundAction{}

// NewNetworkOutboundAction is a constructor
func NewNetworkOutboundAction(collectIntervalInSeconds int, collectCountForProcess int, collectCountForExport int, exporter interfaces.Exporter) interfaces.Action {
	return &networkOutboundAction{
		name:                     "networkoutbound",
		collectIntervalInSeconds: collectIntervalInSeconds,
		collectCountForProcess:   collectCountForProcess,
		collectCountForExport:    collectCountForExport,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *networkOutboundAction) GetName() string {
	return action.name
}

// GetName implements the interface method
func (action *networkOutboundAction) GetCollectIntervalInSeconds() int {
	return action.collectIntervalInSeconds
}

// GetName implements the interface method
func (action *networkOutboundAction) GetCollectCountForProcess() int {
	return action.collectCountForProcess
}

// GetName implements the interface method
func (action *networkOutboundAction) GetCollectCountForExport() int {
	return action.collectCountForExport
}

// Collect implements the interface method
func (action *networkOutboundAction) Collect() error {
	action.collectFiles = []string{}

	APIServerFQDN, err := utils.GetAPIServerFQDN()
	if err != nil {
		return err
	}

	outboundTypes := []networkOutboundType{}
	outboundTypes = append(outboundTypes,
		networkOutboundType{
			Type: "InternetConnectivity",
			URL:  "google.com:80",
		},
	)
	outboundTypes = append(outboundTypes,
		networkOutboundType{
			Type: "APIServerConnectivity",
			URL:  "kubernetes.default.svc.cluster.local:443",
		},
	)
	outboundTypes = append(outboundTypes,
		networkOutboundType{
			Type: "TunnelConnectivity",
			URL:  APIServerFQDN + ":9000",
		},
	)
	outboundTypes = append(outboundTypes,
		networkOutboundType{
			Type: "ACRConnectivity",
			URL:  "azurecr.io:80",
		},
	)
	outboundTypes = append(outboundTypes,
		networkOutboundType{
			Type: "MCRConnectivity",
			URL:  "mcr.microsoft.com:80",
		},
	)
	rootPath, err := utils.CreateCollectorDir(action.name)
	if err != nil {
		return err
	}

	for _, outboundType := range outboundTypes {
		networkOutboundFile := filepath.Join(rootPath, outboundType.Type)

		f, err := os.OpenFile(networkOutboundFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		defer f.Close()
		if err != nil {
			return fmt.Errorf("Fail to open file %s: %+v", networkOutboundFile, err)
		}

		timeout := time.Duration(5 * time.Second)
		_, err = net.DialTimeout("tcp", outboundType.URL, timeout)

		// only write when connection failed
		if err != nil {
			data := &networkOutboundDatum{
				TimeStamp:           time.Now().Truncate(1 * time.Second),
				networkOutboundType: outboundType,
				Connected:           err == nil,
				Error:               err.Error(),
			}

			dataBytes, err := json.Marshal(data)
			if err != nil {
				return fmt.Errorf("Fail to marshal data: %+v", err)
			}

			_, err = f.WriteString(string(dataBytes) + "\n")
			if err != nil {
				return fmt.Errorf("Fail to write data to file: %+v", err)
			}
		}

		action.collectFiles = append(action.collectFiles, networkOutboundFile)
	}

	return nil
}

// Process implements the interface method
func (action *networkOutboundAction) Process() error {
	action.processFiles = []string{}

	rootPath, err := utils.CreateDiagnosticDir()
	if err != nil {
		return err
	}

	networkOutboundDiagnosticFile := filepath.Join(rootPath, action.name)

	f, err := os.OpenFile(networkOutboundDiagnosticFile, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		return fmt.Errorf("Fail to open file %s: %+v", networkOutboundDiagnosticFile, err)
	}

	outboundDiagnosticData := []networkOutboundDiagnosticDatum{}

	for _, file := range action.collectFiles {
		t, err := os.Open(file)
		defer t.Close()
		if err != nil {
			return fmt.Errorf("Fail to open file %s: %+v", file, err)
		}

		dataPoint := networkOutboundDiagnosticDatum{}
		scanner := bufio.NewScanner(t)
		for scanner.Scan() {
			var outboundDatum networkOutboundDatum
			json.Unmarshal([]byte(scanner.Text()), &outboundDatum)

			if dataPoint.Start.IsZero() {
				setDataPoint(&outboundDatum, &dataPoint)
			} else {
				if outboundDatum.Error != dataPoint.Error {
					outboundDiagnosticData = append(outboundDiagnosticData, dataPoint)
					setDataPoint(&outboundDatum, &dataPoint)
				} else {
					if int(outboundDatum.TimeStamp.Sub(dataPoint.End).Seconds()) > action.collectIntervalInSeconds {
						outboundDiagnosticData = append(outboundDiagnosticData, dataPoint)
						setDataPoint(&outboundDatum, &dataPoint)
					} else {
						dataPoint.End = outboundDatum.TimeStamp
					}
				}
			}
		}

		if !dataPoint.Start.IsZero() {
			outboundDiagnosticData = append(outboundDiagnosticData, dataPoint)
		}
	}

	for _, dataPoint := range outboundDiagnosticData {
		dataBytes, err := json.Marshal(dataPoint)
		if err != nil {
			return fmt.Errorf("Fail to marshal data: %+v", err)
		}

		_, err = f.WriteString(string(dataBytes) + "\n")
		if err != nil {
			return fmt.Errorf("Fail to write data to file: %+v", err)
		}
	}

	action.processFiles = append(action.processFiles, networkOutboundDiagnosticFile)

	return nil
}

// Export implements the interface method
func (action *networkOutboundAction) Export() error {
	if action.exporter != nil {
		return action.exporter.Export(append(action.collectFiles, action.processFiles...))
	}

	return nil
}

func setDataPoint(outboundDatum *networkOutboundDatum, dataPoint *networkOutboundDiagnosticDatum) {
	dataPoint.Type = outboundDatum.Type
	dataPoint.Start = outboundDatum.TimeStamp
	dataPoint.End = outboundDatum.TimeStamp
	dataPoint.Error = outboundDatum.Error
}
