package diagnoser

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/aks-periscope/pkg/collector"
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
	runtimeInfo              *utils.RuntimeInfo
	networkOutboundCollector *collector.NetworkOutboundCollector
	data                     map[string]string
}

// NewNetworkOutboundDiagnoser is a constructor
func NewNetworkOutboundDiagnoser(runtimeInfo *utils.RuntimeInfo, networkOutboundCollector *collector.NetworkOutboundCollector) *NetworkOutboundDiagnoser {
	return &NetworkOutboundDiagnoser{
		runtimeInfo:              runtimeInfo,
		networkOutboundCollector: networkOutboundCollector,
		data:                     make(map[string]string),
	}
}

func (collector *NetworkOutboundDiagnoser) GetName() string {
	return "networkoutbound"
}

// Diagnose implements the interface method
func (diagnoser *NetworkOutboundDiagnoser) Diagnose() error {
	outboundDiagnosticData := []networkOutboundDiagnosticDatum{}

	for _, data := range diagnoser.networkOutboundCollector.GetData() {
		dataPoint := networkOutboundDiagnosticDatum{HostName: diagnoser.runtimeInfo.HostNodeName}
		lines := strings.Split(data, "\n")
		for _, line := range lines {
			var outboundDatum collector.NetworkOutboundDatum
			err := json.Unmarshal([]byte(line), &outboundDatum)
			if err != nil {
				log.Printf("Unmarshal failed: %v", err)
				continue
			}

			if dataPoint.Start.IsZero() {
				setDataPoint(&outboundDatum, &dataPoint)
			} else {
				if outboundDatum.Status != dataPoint.Status {
					outboundDiagnosticData = append(outboundDiagnosticData, dataPoint)
					setDataPoint(&outboundDatum, &dataPoint)
				} else {
					if int(outboundDatum.TimeStamp.Sub(dataPoint.End).Seconds()) > 5 {
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
		return fmt.Errorf("marshal data from NetworkOutbound Diagnoser: %w", err)
	}

	diagnoser.data["networkoutbound"] = string(dataBytes)

	return nil
}

func (collector *NetworkOutboundDiagnoser) GetData() map[string]string {
	return collector.data
}

func setDataPoint(outboundDatum *collector.NetworkOutboundDatum, dataPoint *networkOutboundDiagnosticDatum) {
	dataPoint.Type = outboundDatum.Type
	dataPoint.Start = outboundDatum.TimeStamp
	dataPoint.End = outboundDatum.TimeStamp
	dataPoint.Status = outboundDatum.Status
}
