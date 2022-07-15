package main

import (
	"bytes"
	"log"
	"sync"
	"time"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/diagnoser"
	"github.com/Azure/aks-periscope/pkg/exporter"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	restclient "k8s.io/client-go/rest"
)

func main() {
	config, err := restclient.InClusterConfig()
	if err != nil {
		log.Fatalf("Cannot load kubeconfig: %v", err)
	}

	creationTimeStamp, err := utils.GetCreationTimeStamp(config)
	if err != nil {
		log.Fatalf("Failed to get creation timestamp: %v", err)
	}

	runtimeInfo, err := utils.GetRuntimeInfo()
	if err != nil {
		log.Fatalf("Failed to get runtime information: %v", err)
	}

	knownFilePaths, err := utils.GetKnownFilePaths(runtimeInfo)
	if err != nil {
		log.Fatalf("Failed to get file paths: %v", err)
	}

	exp := exporter.NewAzureBlobExporter(runtimeInfo, knownFilePaths, creationTimeStamp)

	// Copies self-signed cert information to container if application is running on Azure Stack Cloud.
	// We need the cert in order to communicate with the storage account.
	if utils.IsAzureStackCloud(knownFilePaths) {
		if err := utils.CopyFile(knownFilePaths.AzureStackCertHost, knownFilePaths.AzureStackCertContainer); err != nil {
			log.Fatalf("Cannot copy cert for Azure Stack Cloud environment: %v", err)
		}
	}

	fileContentReader := utils.NewFileContentReader()

	dnsCollector := collector.NewDNSCollector(runtimeInfo, knownFilePaths, fileContentReader)
	kubeletCmdCollector := collector.NewKubeletCmdCollector(runtimeInfo)
	networkOutboundCollector := collector.NewNetworkOutboundCollector()
	collectors := []interfaces.Collector{
		dnsCollector,
		kubeletCmdCollector,
		networkOutboundCollector,
		collector.NewHelmCollector(config, runtimeInfo),
		collector.NewIPTablesCollector(runtimeInfo),
		collector.NewKubeObjectsCollector(config, runtimeInfo),
		collector.NewNodeLogsCollector(runtimeInfo, fileContentReader),
		collector.NewOsmCollector(config, runtimeInfo),
		collector.NewPDBCollector(config, runtimeInfo),
		collector.NewPodsContainerLogsCollector(config, runtimeInfo),
		collector.NewSmiCollector(config, runtimeInfo),
		collector.NewSystemLogsCollector(runtimeInfo),
		collector.NewSystemPerfCollector(config, runtimeInfo),
		collector.NewWindowsLogsCollector(runtimeInfo, knownFilePaths, fileContentReader, 10*time.Second, 20*time.Minute),
	}

	collectorGrp := new(sync.WaitGroup)

	dataProducers := []interfaces.DataProducer{}
	for _, c := range collectors {
		if err := c.CheckSupported(); err != nil {
			// Log the reason why this collector is not supported, and skip to the next
			log.Printf("Skipping unsupported collector %s: %v", c.GetName(), err)
			continue
		}

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
		diagnoser.NewNetworkConfigDiagnoser(runtimeInfo, dnsCollector, kubeletCmdCollector),
		diagnoser.NewNetworkOutboundDiagnoser(runtimeInfo, networkOutboundCollector),
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
		if err := exp.ExportReader(runtimeInfo.HostNodeName+".zip", bytes.NewReader(zip.Bytes())); err != nil {
			log.Printf("Could not export zip archive: %v", err)
		}
	}

	// TODO: Hack: for now AKS-Periscope is running as a deamonset so it shall not stop (or the pod will be restarted)
	// Revert from https://github.com/Azure/aks-periscope/blob/b98d66a238e942158ef2628a9315b58937ff9c8f/cmd/aks-periscope/aks-periscope.go#L70
	select {}

	// TODO: remove this //nolint comment once the select{} has been removed
	//nolint:govet
}
