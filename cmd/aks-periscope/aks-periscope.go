package main

import (
	"bytes"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/diagnoser"
	"github.com/Azure/aks-periscope/pkg/exporter"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	restclient "k8s.io/client-go/rest"
)

func main() {
	creationTimeStamp, err := utils.GetCreationTimeStamp()
	if err != nil {
		log.Fatalf("Failed to get creation timestamp: %v", err)
	}

	hostname, err := utils.GetHostName()
	if err != nil {
		log.Fatalf("Failed to get the hostname on which AKS Periscope is running: %v", err)
	}

	if err := utils.CreateCRD(); err != nil {
		log.Fatalf("Failed to create CRD: %v", err)
	}

	collectorList := strings.Fields(os.Getenv("COLLECTOR_LIST"))
	exp := exporter.NewAzureBlobExporter(creationTimeStamp, hostname)

	// Copies self-signed cert information to container if application is running on Azure Stack Cloud.
	// We need the cert in order to communicate with the storage account.
	if utils.IsAzureStackCloud() {
		if err := utils.CopyFileFromHost("/etc/ssl/certs/azsCertificate.pem", "/etc/ssl/certs/azsCertificate.pem"); err != nil {
			log.Fatalf("Cannot copy cert for Azure Stack Cloud environment: %v", err)
		}
	}

	config, err := restclient.InClusterConfig()
	if err != nil {
		log.Fatalf("Cannot load kubeconfig: %v", err)
	}

	dataProducers := []interfaces.DataProducer{}

	containerLogsCollector := collector.NewContainerLogsCollector()
	networkOutboundCollector := collector.NewNetworkOutboundCollector()
	dnsCollector := collector.NewDNSCollector()
	kubeObjectsCollector := collector.NewKubeObjectsCollector()
	systemLogsCollector := collector.NewSystemLogsCollector()
	ipTablesCollector := collector.NewIPTablesCollector()
	nodeLogsCollector := collector.NewNodeLogsCollector()
	kubeletCmdCollector := collector.NewKubeletCmdCollector()
	systemPerfCollector := collector.NewSystemPerfCollector()
	helmCollector := collector.NewHelmCollector(config)
	osmCollector := collector.NewOsmCollector()
	smiCollector := collector.NewSmiCollector()

	collectors := []interfaces.Collector{
		containerLogsCollector,
		dnsCollector,
		kubeObjectsCollector,
		networkOutboundCollector,
	}

	if contains(collectorList, "connectedCluster") {
		collectors = append(collectors, helmCollector)
	} else {
		collectors = append(collectors, systemLogsCollector)
		collectors = append(collectors, ipTablesCollector)
		collectors = append(collectors, nodeLogsCollector)
		collectors = append(collectors, kubeletCmdCollector)
		collectors = append(collectors, systemPerfCollector)
	}

	// OSM and SMI flags are mutually exclusive
	if contains(collectorList, "OSM") {
		collectors = append(collectors, osmCollector)
		collectors = append(collectors, smiCollector)
	} else if contains(collectorList, "SMI") {
		collectors = append(collectors, smiCollector)
	}

	collectorGrp := new(sync.WaitGroup)

	for _, c := range collectors {
		dataProducers = append(dataProducers, c)
		collectorGrp.Add(1)
		go func(c interfaces.Collector) {
			defer collectorGrp.Done()

			log.Printf("Collector: %s, collect data", c.GetName())
			err := c.Collect()
			if err != nil {
				log.Printf("Collector: %s, collect data failed: %v", c.GetName(), err)
				return
			}

			log.Printf("Collector: %s, export data", c.GetName())
			if err = exp.Export(c); err != nil {
				log.Printf("Collector: %s, export data failed: %v", c.GetName(), err)
			}
		}(c)
	}

	collectorGrp.Wait()

	diagnosers := []interfaces.Diagnoser{
		diagnoser.NewNetworkConfigDiagnoser(dnsCollector, kubeletCmdCollector),
		diagnoser.NewNetworkOutboundDiagnoser(networkOutboundCollector),
	}

	diagnoserGrp := new(sync.WaitGroup)

	for _, d := range diagnosers {
		dataProducers = append(dataProducers, d)
		diagnoserGrp.Add(1)
		go func(d interfaces.Diagnoser) {
			defer diagnoserGrp.Done()

			log.Printf("Diagnoser: %s, diagnose data", d.GetName())
			err := d.Diagnose()
			if err != nil {
				log.Printf("Diagnoser: %s, diagnose data failed: %v", d.GetName(), err)
				return
			}

			log.Printf("Diagnoser: %s, export data", d.GetName())
			if err = exp.Export(d); err != nil {
				log.Printf("Diagnoser: %s, export data failed: %v", d.GetName(), err)
			}
		}(d)
	}

	diagnoserGrp.Wait()

	zip, err := exporter.Zip(dataProducers)
	if err != nil {
		log.Printf("Could not zip data: %v", err)
	} else {
		if err := exp.ExportReader(hostname+".zip", bytes.NewReader(zip.Bytes())); err != nil {
			log.Printf("Could not export zip archive: %v", err)
		}
	}

	// TODO: Hack: for now AKS-Periscope is running as a deamonset so it shall not stop (or the pod will be restarted)
	// Revert from https://github.com/Azure/aks-periscope/blob/b98d66a238e942158ef2628a9315b58937ff9c8f/cmd/aks-periscope/aks-periscope.go#L70
	select {}

	// TODO: remove this //nolint comment once the select{} has been removed
	//nolint:govet
}

func contains(flagsList []string, flag string) bool {
	for _, f := range flagsList {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}
