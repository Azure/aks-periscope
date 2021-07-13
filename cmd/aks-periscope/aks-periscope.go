package main

import (
	"bytes"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/diagnoser"
	"github.com/Azure/aks-periscope/pkg/exporter"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"

	"github.com/hashicorp/go-multierror"
)

func main() {
	creationTimeStamp, err := utils.GetCreationTimeStamp()
	if err != nil {
		log.Fatalf("Failed to get creation timestamp: %v", err)
	}
	log.Printf("Setting creation timestamp to %s", creationTimeStamp)

	hostname, err := utils.GetHostName()
	if err != nil {
		log.Fatalf("Failed to get the hostname on which AKS Periscope is running: %v", err)
	}
	log.Printf("Setting hostname to %s", hostname)

	if err := utils.CreateCRD(); err != nil {
		log.Fatalf("Failed to create CRD: %v", err)
	}

	// Copies self-signed cert information to container if application is running on Azure Stack Cloud.
	// We need the cert in order to communicate with the storage account.
	if utils.IsAzureStackCloud() {
		if err := utils.CopyFileFromHost("/etc/ssl/certs/azsCertificate.pem", "/etc/ssl/certs/azsCertificate.pem"); err != nil {
			log.Fatalf("cannot copy cert for Azure Stack Cloud environment: %v", err)
		}
	}

	collectors, diagnosers, exporters := initializeComponents(creationTimeStamp, hostname)

	//dataProducers includes all selected collectors and diagnosers
	dataProducers := []interfaces.DataProducer{}
	for _, c := range collectors {
		dataProducers = append(dataProducers, c)
	}
	for _, d := range diagnosers {
		dataProducers = append(dataProducers, d)
	}

	collectorGrp := new(sync.WaitGroup)
	runCollectors(collectors, exporters, collectorGrp)
	collectorGrp.Wait()

	diagnoserGrp := new(sync.WaitGroup)
	runDiagnosers(diagnosers, exporters, diagnoserGrp)
	diagnoserGrp.Wait()

	zipAndExportString, found := os.LookupEnv("ZIP_AND_EXPORT")
	zipAndExport, parseErr := strconv.ParseBool(zipAndExportString)
	if !found || parseErr != nil {
		zipAndExport = true
	}

	if zipAndExport {
		log.Print("Zip result files")
		zip, err := exporter.Zip(dataProducers)
		if err != nil {
			log.Printf("Could not zip data: %v", err)
		} else {

			if err := runExportersForZip(exporters, zip, hostname); err != nil {
				log.Printf("Could not export zip archive: %v", err)
			}
		}
	}

	// TODO: Hack: for now AKS-Periscope is running as a deamonset so it shall not stop (or the pod will be restarted)
	// Revert from https://github.com/Azure/aks-periscope/blob/b98d66a238e942158ef2628a9315b58937ff9c8f/cmd/aks-periscope/aks-periscope.go#L70
	select {}
}

// initializeComponents initializes and returns collectors, diagnosers and exporters to run
// first it creates an instance of each type, and then it calls the relevant select* method (selectCollector, selectDiagnoser, selectExporter)
// passing in a map of instance name -> instance object, and getting back a filtered list of just the selected components
func initializeComponents(creationTimeStamp string, hostname string) ([]interfaces.Collector, []interfaces.Diagnoser, []interfaces.Exporter) {
	//TODO it would be nice if we only instantiated those collector/diagnoser/exporters that were actually selected for execution

	//collectors
	containerLogsCollector := collector.NewContainerLogsCollector()
	networkOutboundCollector := collector.NewNetworkOutboundCollector()
	dnsCollector := collector.NewDNSCollector()
	kubeObjectsCollector := collector.NewKubeObjectsCollector()
	systemLogsCollector := collector.NewSystemLogsCollector()
	ipTablesCollector := collector.NewIPTablesCollector()
	nodeLogsCollector := collector.NewNodeLogsCollector()
	kubeletCmdCollector := collector.NewKubeletCmdCollector()
	systemPerfCollector := collector.NewSystemPerfCollector()
	helmCollector := collector.NewHelmCollector()
	osmCollector := collector.NewOsmCollector()

	selectedCollectors := selectCollectors(
		map[string]interfaces.Collector{
			containerLogsCollector.GetName():           containerLogsCollector,
			systemLogsCollector.GetName():              systemLogsCollector,
			networkOutboundCollector.GetName():         networkOutboundCollector,
			ipTablesCollector.GetName():                ipTablesCollector,
			nodeLogsCollector.GetName():                nodeLogsCollector,
			dnsCollector.GetName():                     dnsCollector,
			kubeObjectsCollector.GetName():             kubeObjectsCollector,
			kubeletCmdCollector.GetName():              kubeletCmdCollector,
			systemPerfCollector.GetName():              systemPerfCollector,
			helmCollector.GetName():                    helmCollector,
			osmCollector.GetName():                     osmCollector,
		})

	//diagnosers
	//NOTE currently the collector instance objects are shared between the collector itself and things which use it as a dependency
	networkConfigDiagnoser := diagnoser.NewNetworkConfigDiagnoser(hostname, dnsCollector, kubeletCmdCollector)
	networkOutboundDiagnoser := diagnoser.NewNetworkOutboundDiagnoser(hostname, networkOutboundCollector)
	selectedDiagnosers := selectDiagnosers(
		map[string]interfaces.Diagnoser{
			networkConfigDiagnoser.GetName():   networkConfigDiagnoser,
			networkOutboundDiagnoser.GetName(): networkOutboundDiagnoser,
		})

	//exporters
	azureBlobExporter := exporter.NewAzureBlobExporter(creationTimeStamp, hostname)
	selectedExporters := selectExporters(
		map[string]interfaces.Exporter{
			azureBlobExporter.GetName():    azureBlobExporter,
		})

	return selectedCollectors, selectedDiagnosers, selectedExporters
}

// selectCollectors select the collectors to run by looking up the value from the config
func selectCollectors(allCollectorsByName map[string]interfaces.Collector) []interfaces.Collector {
	collectors := []interfaces.Collector{}

	enabledCollectorString, found := os.LookupEnv("ENABLED_COLLECTORS")
	if !found {
		//if not defined, default to all collectors enabled
		enabledCollectorString = "containerlogs dns helm iptables kubeletcmd kubeobjects networkoutbound nodelogs osm systemlogs systemperf"
	}

	enabledCollectorNames := strings.Fields(enabledCollectorString)

	for _, collectorName := range enabledCollectorNames {
		collectors = append(collectors, allCollectorsByName[collectorName])
	}

	return collectors
}

// selectDiagnosers select the diagnosers to run
func selectDiagnosers(allDiagnosersByName map[string]interfaces.Diagnoser) []interfaces.Diagnoser {
	diagnosers := []interfaces.Diagnoser{}

	//read list of diagnosers that are enabled
	enabledDiagnoserString, found := os.LookupEnv("ENABLED_DIAGNOSERS")
	if !found {
		//if not defined, default to all diagnosers enabled
		enabledDiagnoserString = "networkconfig networkoutbound"
	}

	enabledDiagnoserNames := strings.Fields(enabledDiagnoserString)

	for _, diagnoserName := range enabledDiagnoserNames {
		diagnosers = append(diagnosers, allDiagnosersByName[diagnoserName])
	}

	return diagnosers
}

// selectedExporters select the exporters to run
func selectExporters(allExporters map[string]interfaces.Exporter) []interfaces.Exporter {
	exporters := []interfaces.Exporter{}

	//read list of exporters that are enabled
	enabledExportersString, found := os.LookupEnv("ENABLED_EXPORTERS")
	if !found {
		//if not defined, default to all exporters enabled
		enabledExportersString = "azureblob"
	}

	enabledExporterNames := strings.Fields(enabledExportersString)

	for _, exporterName := range enabledExporterNames {
		exporters = append(exporters, allExporters[exporterName])
	}

	return exporters
}

// runCollectors run the collectors
func runCollectors(collectors []interfaces.Collector, exporters []interfaces.Exporter, collectorGrp *sync.WaitGroup) {
	for _, c := range collectors {
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
			if err = runExportersForDataProducer(exporters, c); err != nil {
				log.Printf("Collector: %s, export data failed: %v", c.GetName(), err)
			}
		}(c)
	}
}

// runDiagnosers run the diagnosers
func runDiagnosers(diagnosers []interfaces.Diagnoser, exporters []interfaces.Exporter, diagnoserGrp *sync.WaitGroup) {
	for _, d := range diagnosers {
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
			if err = runExportersForDataProducer(exporters, d); err != nil {
				log.Printf("Diagnoser: %s, export data failed: %v", d.GetName(), err)
			}
		}(d)
	}
}

// runExporters run the exporters for a data producer
func runExportersForDataProducer(exporters []interfaces.Exporter, dataProducer interfaces.DataProducer) error {
	var result error
	for _, e := range exporters {
		if err := e.Export(dataProducer); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}

//runExportersForZip run the exporters for the zip file
func runExportersForZip(exporters []interfaces.Exporter, zip *bytes.Buffer, hostname string) error {
	var result error
	for _, exp := range exporters {
		if err := exp.ExportReader(hostname+".zip", bytes.NewReader(zip.Bytes())); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}
