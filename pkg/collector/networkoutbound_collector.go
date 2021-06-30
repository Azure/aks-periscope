package collector

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/Azure/aks-periscope/pkg/utils"
)

type networkOutboundType struct {
	Type string `json:"Type"`
	URL  string `json:"URL"`
}

// NetworkOutboundDatum defines a NetworkOutbound Datum
type NetworkOutboundDatum struct {
	TimeStamp time.Time `json:"TimeStamp"`
	networkOutboundType
	Status string `json:"Status"`
}

// NetworkOutboundCollector defines a NetworkOutbound Collector struct
type NetworkOutboundCollector struct {
	data map[string]string
}

// NewNetworkOutboundCollector is a constructor
func NewNetworkOutboundCollector() *NetworkOutboundCollector {
	return &NetworkOutboundCollector{
		data: make(map[string]string),
	}
}

func (collector *NetworkOutboundCollector) GetName() string {
	return "networkoutbound"
}

// Collect implements the interface method
func (collector *NetworkOutboundCollector) Collect() error {
	APIServerFQDN, err := utils.GetAPIServerFQDN()
	if err != nil {
		return err
	}

	outboundTypes := []networkOutboundType{}
	outboundTypes = append(outboundTypes,
		networkOutboundType{
			Type: "Internet",
			URL:  "google.com:80",
		},
	)
	outboundTypes = append(outboundTypes,
		networkOutboundType{
			Type: "AKS API Server",
			URL:  "kubernetes.default.svc.cluster.local:443",
		},
	)
	outboundTypes = append(outboundTypes,
		networkOutboundType{
			Type: "AKS Tunnel",
			URL:  APIServerFQDN + ":9000",
		},
	)
	outboundTypes = append(outboundTypes,
		networkOutboundType{
			Type: "Azure Container Registry",
			URL:  "azurecr.io:80",
		},
	)
	outboundTypes = append(outboundTypes,
		networkOutboundType{
			Type: "Microsoft Container Registry",
			URL:  "mcr.microsoft.com:80",
		},
	)

	for _, outboundType := range outboundTypes {
		timeout := time.Duration(5 * time.Second)
		_, err = net.DialTimeout("tcp", outboundType.URL, timeout)

		status := "Connected"
		if err != nil {
			status = "Error: " + err.Error()
		}

		data := &NetworkOutboundDatum{
			TimeStamp:           time.Now().Truncate(1 * time.Second),
			networkOutboundType: outboundType,
			Status:              status,
		}

		dataBytes, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal data: %w", err)
		}

		collector.data[outboundType.Type] = string(dataBytes)
	}

	return nil
}

func (collector *NetworkOutboundCollector) GetData() map[string]string {
	return collector.data
}
