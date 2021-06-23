package collector

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/aks-periscope/pkg/interfaces"
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
	BaseCollector
}

var _ interfaces.Collector = &NetworkOutboundCollector{}

// NewNetworkOutboundCollector is a constructor
func NewNetworkOutboundCollector(collectIntervalInSeconds int, exporters []interfaces.Exporter) *NetworkOutboundCollector {
	return &NetworkOutboundCollector{
		BaseCollector: BaseCollector{
			collectorType:            NetworkOutbound,
			collectIntervalInSeconds: collectIntervalInSeconds,
			exporters:                exporters,
		},
	}
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
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	for _, outboundType := range outboundTypes {
		networkOutboundFile := filepath.Join(rootPath, outboundType.Type)

		f, err := os.OpenFile(networkOutboundFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("Fail to open file %s: %+v", networkOutboundFile, err)
		}
		defer f.Close()

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
			return fmt.Errorf("Fail to marshal data: %+v", err)
		}

		_, err = f.WriteString(string(dataBytes) + "\n")
		if err != nil {
			return fmt.Errorf("Fail to write data to file: %+v", err)
		}

		collector.AddToCollectorFiles(networkOutboundFile)
	}

	return nil
}
