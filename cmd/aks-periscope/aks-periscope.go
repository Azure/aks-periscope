package main

import (
	"bytes"
	"fmt"
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
	// Create a watcher that checks runtime configuration every 10 seconds
	runtimeInfoWatcher := utils.NewRuntimeInfoWatcher(10 * time.Second)

	runtimeInfoChan := make(chan *utils.RuntimeInfo)
	runtimeInfoWatcher.AddHandler(runtimeInfoChan)

	errChan := make(chan error)
	go func() {
		lastRunId := ""
		// Continually check each runtime configuration.
		for {
			runtimeInfo := <-runtimeInfoChan
			// If the run ID has changed, run Periscope
			if runtimeInfo.RunId != lastRunId {
				lastRunId = runtimeInfo.RunId

				log.Printf("Starting Periscope run %s", runtimeInfo.RunId)
				err := run(runtimeInfo)
				if err != nil {
					errChan <- err
				}

				log.Printf("Completed Periscope run %s", runtimeInfo.RunId)
			}
		}
	}()

	// Run until error
	runtimeInfoWatcher.Start()
	err := <-errChan
	log.Fatalf("Error running Periscope: %v", err)
}

func run(runtimeInfo *utils.RuntimeInfo) error {
	config, err := restclient.InClusterConfig()
	if err != nil {
		return fmt.Errorf("cannot load kubeconfig: %w", err)
	}

	knownFilePaths, err := utils.GetKnownFilePaths(runtimeInfo)
	if err != nil {
		return fmt.Errorf("failed to get file paths: %w", err)
	}

	exp := exporter.NewAzureBlobExporter(runtimeInfo, knownFilePaths, runtimeInfo.RunId)

	// Copies self-signed cert information to container if application is running on Azure Stack Cloud.
	// We need the cert in order to communicate with the storage account.
	if utils.IsAzureStackCloud(knownFilePaths) {
		if err := utils.CopyFile(knownFilePaths.AzureStackCertHost, knownFilePaths.AzureStackCertContainer); err != nil {
			return fmt.Errorf("cannot copy cert for Azure Stack Cloud environment: %w", err)
		}
	}

	fileSystem := utils.NewFileSystem()

	dnsCollector := collector.NewDNSCollector(runtimeInfo, knownFilePaths, fileSystem)
	kubeletCmdCollector := collector.NewKubeletCmdCollector(runtimeInfo)
	networkOutboundCollector := collector.NewNetworkOutboundCollector()
	collectors := []interfaces.Collector{
		dnsCollector,
		kubeletCmdCollector,
		networkOutboundCollector,
		collector.NewHelmCollector(config, runtimeInfo),
		collector.NewIPTablesCollector(runtimeInfo),
		collector.NewKubeObjectsCollector(config, runtimeInfo),
		collector.NewNodeLogsCollector(runtimeInfo, fileSystem),
		collector.NewOsmCollector(config, runtimeInfo),
		collector.NewPDBCollector(config, runtimeInfo),
		collector.NewPodsContainerLogsCollector(config, runtimeInfo),
		collector.NewSmiCollector(config, runtimeInfo),
		collector.NewSystemLogsCollector(runtimeInfo),
		collector.NewSystemPerfCollector(config, runtimeInfo),
		collector.NewWindowsLogsCollector(runtimeInfo, knownFilePaths, fileSystem, 10*time.Second, 20*time.Minute),
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

	return nil
}
