package main

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
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
	osIdentifier, err := utils.StringToOSIdentifier(runtime.GOOS)
	if err != nil {
		log.Fatalf("cannot determine OS: %v", err)
	}

	knownFilePaths, err := utils.GetKnownFilePaths(osIdentifier)
	if err != nil {
		log.Fatalf("failed to get file paths: %v", err)
	}

	fileSystem := utils.NewFileSystem()

	// Create a watcher for the Run ID file that checks its content every 10 seconds
	fileWatcher := utils.NewFileContentWatcher(fileSystem, 10*time.Second)

	// Create a channel for unrecoverable errors
	errChan := make(chan error)

	// Add a watcher for the run ID file content
	runIdChan := make(chan string)
	fileWatcher.AddHandler(knownFilePaths.GetConfigPath(utils.RunIdKey), runIdChan, errChan)

	go func() {
		for {
			runId := <-runIdChan
			log.Printf("Starting Periscope run %s", runId)
			err := run(osIdentifier, knownFilePaths, fileSystem)
			if err != nil {
				errChan <- err
			}

			log.Printf("Completed Periscope run %s", runId)
		}
	}()

	fileWatcher.Start()

	// Run until unrecoverable error
	err = <-errChan
	log.Fatalf("Error running Periscope: %v", err)
}

func run(osIdentifier utils.OSIdentifier, knownFilePaths *utils.KnownFilePaths, fileSystem interfaces.FileSystemAccessor) error {
	runtimeInfo, err := utils.GetRuntimeInfo(fileSystem, knownFilePaths)
	if err != nil {
		log.Fatalf("Failed to get runtime information: %v", err)
	}

	config, err := restclient.InClusterConfig()
	if err != nil {
		return fmt.Errorf("cannot load kubeconfig: %w", err)
	}

	exp := exporter.NewAzureBlobExporter(runtimeInfo, knownFilePaths, runtimeInfo.RunId)

	// Copies self-signed cert information to container if application is running on Azure Stack Cloud.
	// We need the cert in order to communicate with the storage account.
	if utils.IsAzureStackCloud(knownFilePaths) {
		if err := utils.CopyFile(knownFilePaths.AzureStackCertHost, knownFilePaths.AzureStackCertContainer); err != nil {
			return fmt.Errorf("cannot copy cert for Azure Stack Cloud environment: %w", err)
		}
	}

	// those collectors are reused in the diagnosers, hence declared explicitly
	dnsCollector := collector.NewDNSCollector(osIdentifier, knownFilePaths, fileSystem)
	kubeletCmdCollector := collector.NewKubeletCmdCollector(osIdentifier, runtimeInfo)
	networkOutboundCollector := collector.NewNetworkOutboundCollector()

	collectors := []interfaces.Collector{
		dnsCollector,
		kubeletCmdCollector,
		networkOutboundCollector,
		collector.NewHelmCollector(config, runtimeInfo),
		collector.NewIPTablesCollector(osIdentifier, runtimeInfo),
		collector.NewKubeObjectsCollector(config, runtimeInfo),
		collector.NewNodeLogsCollector(runtimeInfo, fileSystem),
		collector.NewOsmCollector(config, runtimeInfo),
		collector.NewPDBCollector(config, runtimeInfo),
		collector.NewPodsContainerLogsCollector(config, runtimeInfo),
		collector.NewSmiCollector(config, runtimeInfo),
		collector.NewSystemLogsCollector(osIdentifier, runtimeInfo),
		collector.NewSystemPerfCollector(config, runtimeInfo),
		collector.NewWindowsLogsCollector(osIdentifier, runtimeInfo, knownFilePaths, fileSystem, 10*time.Second, 20*time.Minute),
	}

	collectorGrp := new(sync.WaitGroup)

	var dataProducers []interfaces.DataProducer

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
