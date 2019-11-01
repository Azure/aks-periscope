package diagnoser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

type networkOutboundDiagnosticDatum struct {
	HostName string    `json:"HostName"`
	Type     string    `json:"Type"`
	Start    time.Time `json:"Start"`
	End      time.Time `json:"End"`
	Status   string    `json:"Status"`
}

// NetworkOutboundDiagnoser defines a NetworkOutbound Diagnoser struct
type NetworkOutboundDiagnoser struct {
	BaseDiagnoser
	networkOutboundCollector *collector.NetworkOutboundCollector
}

var _ interfaces.Diagnoser = &NetworkOutboundDiagnoser{}

// NewNetworkOutboundDiagnoser is a constructor
func NewNetworkOutboundDiagnoser(networkOutboundCollector *collector.NetworkOutboundCollector, exporter interfaces.Exporter) *NetworkOutboundDiagnoser {
	return &NetworkOutboundDiagnoser{
		BaseDiagnoser: BaseDiagnoser{
			diagnoserType: NetworkOutbound,
			exporter:      exporter,
		},
		networkOutboundCollector: networkOutboundCollector,
	}
}

// Diagnose implements the interface method
func (diagnoser *NetworkOutboundDiagnoser) Diagnose() error {
	hostName, err := utils.GetHostName()
	rootPath, err := utils.CreateDiagnosticDir()
	if err != nil {
		return err
	}

	networkOutboundDiagnosticFile := filepath.Join(rootPath, diagnoser.GetName())

	f, err := os.OpenFile(networkOutboundDiagnosticFile, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		return fmt.Errorf("Fail to open file %s: %+v", networkOutboundDiagnosticFile, err)
	}

	outboundDiagnosticData := []networkOutboundDiagnosticDatum{}

	for _, file := range diagnoser.networkOutboundCollector.GetCollectorFiles() {
		t, err := os.Open(file)
		defer t.Close()
		if err != nil {
			return fmt.Errorf("Fail to open file %s: %+v", file, err)
		}

		dataPoint := networkOutboundDiagnosticDatum{HostName: hostName}
		scanner := bufio.NewScanner(t)
		for scanner.Scan() {
			var outboundDatum collector.NetworkOutboundDatum
			json.Unmarshal([]byte(scanner.Text()), &outboundDatum)

			if dataPoint.Start.IsZero() {
				setDataPoint(&outboundDatum, &dataPoint)
			} else {
				if outboundDatum.Status != dataPoint.Status {
					outboundDiagnosticData = append(outboundDiagnosticData, dataPoint)
					setDataPoint(&outboundDatum, &dataPoint)
				} else {
					if int(outboundDatum.TimeStamp.Sub(dataPoint.End).Seconds()) > diagnoser.networkOutboundCollector.GetCollectIntervalInSeconds() {
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

	dataBytes, err := json.Marshal(outboundDiagnosticData)
	if err != nil {
		return fmt.Errorf("Fail to marshal data: %+v", err)
	}

	_, err = f.WriteString(string(dataBytes) + "\n")
	if err != nil {
		return fmt.Errorf("Fail to write data to file: %+v", err)
	}

	diagnoser.AddToDiagnoserFiles(networkOutboundDiagnosticFile)

	err = utils.WriteToCRD(networkOutboundDiagnosticFile, diagnoser.GetName())
	if err != nil {
		return fmt.Errorf("Fail to write file %s to CRD: %+v", networkOutboundDiagnosticFile, err)
	}

	return nil
}

func setDataPoint(outboundDatum *collector.NetworkOutboundDatum, dataPoint *networkOutboundDiagnosticDatum) {
	dataPoint.Type = outboundDatum.Type
	dataPoint.Start = outboundDatum.TimeStamp
	dataPoint.End = outboundDatum.TimeStamp
	dataPoint.Status = outboundDatum.Status
}
