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
	Connected bool   `json:"Connected"`
	Error     string `json:"Error"`
}

// NetworkOutboundCollector defines a NetworkOutbound Collector struct
type NetworkOutboundCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &NetworkOutboundCollector{}

// NewNetworkOutboundCollector is a constructor
func NewNetworkOutboundCollector(collectIntervalInSeconds int, exporter interfaces.Exporter) *NetworkOutboundCollector {
	return &NetworkOutboundCollector{
		BaseCollector: BaseCollector{
			collectorType:            NetworkOutbound,
			collectIntervalInSeconds: collectIntervalInSeconds,
			exporter:                 exporter,
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
	outboundTypes = append(outboundTypes,
		networkOutboundType{
			Type: "NotReachableSite",
			URL:  "www.notreachable.site:80",
		},
	)
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
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
			data := &NetworkOutboundDatum{
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

		collector.AddToCollectorFiles(networkOutboundFile)
	}

	return nil
}
