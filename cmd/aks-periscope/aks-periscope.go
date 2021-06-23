package main

import (
	"log"
	"os"
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

	err := utils.CreateCRD()
	if err != nil {
		log.Fatalf("Failed to create CRD: %v", err)
	}

	collectorList := strings.Fields(os.Getenv("COLLECTOR_LIST"))

	// Copies self-signed cert information to container if application is running on Azure Stack Cloud.
	// We need the cert in order to communicate with the storage account.
	if utils.IsAzureStackCloud() {
		if err := utils.CopyFileFromHost("/etc/ssl/certs/azsCertificate.pem", "/etc/ssl/certs/azsCertificate.pem"); err != nil {
			log.Fatalf("cannot copy cert for Azure Stack Cloud environment: %v", err)
		}
	}

	collectors, diagnosers, exporters := initializeComponents()

	collectorGrp := new(sync.WaitGroup)
	runCollectors(collectors, collectorGrp)
	collectorGrp.Wait()

	diagnoserGrp := new(sync.WaitGroup)
	runDiagnosers(diagnosers, diagnoserGrp)
	diagnoserGrp.Wait()

	log.Print("Zip result files")
	zippedOutputs, err := zipOutputDirectory()
	if err != nil {
		log.Printf("Failed to zip result files: %+v", err)
	}

	log.Print("Run exporters for result files")
	err = runExporters(exporters, zippedOutputs)
	if err != nil {
		log.Printf("Failed to export result files: %+v", err)
	}

	// TODO: Hack: for now AKS-Periscope is running as a deamonset so it shall not stop (or the pod will be restarted)
	// Revert from https://github.com/Azure/aks-periscope/blob/b98d66a238e942158ef2628a9315b58937ff9c8f/cmd/aks-periscope/aks-periscope.go#L70
	select {}

	// TODO: remove this //nolint comment once the select{} has been removed
	//nolint:govet
	return
}

// initializeComponents initializes and returns collectors, diagnosers and exporters
func initializeComponents() ([]interfaces.Collector, []interfaces.Diagnoser, []interfaces.Exporter) {

	//exporters
	azureBlobExporter := exporter.NewAzureBlobExporter()
	selectedExporters := selectExporters(
		map[string]interfaces.Exporter{
			azureBlobExporter.GetName(): azureBlobExporter,
		})

	//collectors
	containerLogsCollector := collector.NewContainerLogsCollector(selectedExporters)
	systemLogsCollector := collector.NewSystemLogsCollector(selectedExporters)
	networkOutboundCollector := collector.NewNetworkOutboundCollector(5, selectedExporters)
	ipTablesCollector := collector.NewIPTablesCollector(selectedExporters)
	dnsCollector := collector.NewDNSCollector(selectedExporters)
	nodeLogsCollector := collector.NewNodeLogsCollector(selectedExporters)
	kubeObjectsCollector := collector.NewKubeObjectsCollector(selectedExporters)
	kubeletCmdCollector := collector.NewKubeletCmdCollector(selectedExporters)
	systemPerfCollector := collector.NewSystemPerfCollector(selectedExporters)
	helmCollector := collector.NewHelmCollector(selectedExporters)
	osmCollector := collector.NewOsmCollector(selectedExporters)

	selectedCollectors := selectCollectors(
		map[string]interfaces.Collector{
			containerLogsCollector.GetName():   containerLogsCollector,
			systemLogsCollector.GetName():      systemLogsCollector,
			networkOutboundCollector.GetName(): networkOutboundCollector,
			ipTablesCollector.GetName():        ipTablesCollector,
			nodeLogsCollector.GetName():        nodeLogsCollector,
			dnsCollector.GetName():             dnsCollector,
			kubeObjectsCollector.GetName():     kubeObjectsCollector,
			kubeletCmdCollector.GetName():      kubeletCmdCollector,
			systemPerfCollector.GetName():      systemPerfCollector,
			helmCollector.GetName():            helmCollector,
			osmCollector.GetName():             osmCollector,
		})

	//diagnosers
	//NOTE currently the collector instance objects are shared between the collector itself and things which use it as a dependency
	networkConfigDiagnoser := diagnoser.NewNetworkConfigDiagnoser(dnsCollector, kubeletCmdCollector, selectedExporters)
	networkOutboundDiagnoser := diagnoser.NewNetworkOutboundDiagnoser(networkOutboundCollector, selectedExporters)
	selectedDiagnosers := selectDiagnosers(
		map[string]interfaces.Diagnoser{
			networkConfigDiagnoser.GetName():   networkConfigDiagnoser,
			networkOutboundDiagnoser.GetName(): networkOutboundDiagnoser,
		})

	return selectedCollectors, selectedDiagnosers, selectedExporters
}

// selectCollectors select the collectors to run
func selectCollectors(allCollectorsByName map[string]interfaces.Collector) []interfaces.Collector {
	collectors := []interfaces.Collector{}

	collectors = append(collectors, containerLogsCollector)
	collectors = append(collectors, dnsCollector)
	collectors = append(collectors, kubeObjectsCollector)
	collectors = append(collectors, networkOutboundCollector)

	if contains(collectorList, "connectedCluster") {
		collectors = append(collectors, helmCollector)
	} else {
		collectors = append(collectors, systemLogsCollector)
		collectors = append(collectors, ipTablesCollector)
		collectors = append(collectors, nodeLogsCollector)
		collectors = append(collectors, kubeletCmdCollector)
		collectors = append(collectors, systemPerfCollector)
	}

	if contains(collectorList, "OSM") {
		collectors = append(collectors, osmCollector)
	}

	//read list of collectors that are enabled
	enabledCollectorNames := strings.Fields(os.Getenv("ENABLED_COLLECTORS"))

	for _, collector := range enabledCollectorNames {
		collectors = append(collectors, allCollectorsByName[collector])
	}

	return collectors
}

// selectDiagnosers select the diagnosers to run
func selectDiagnosers(allDiagnosersByName map[string]interfaces.Diagnoser) []interfaces.Diagnoser {
	diagnosers := []interfaces.Diagnoser{}

	//read list of diagnosers that are enabled
	enabledDiagnoserNames := strings.Fields(os.Getenv("ENABLED_DIAGNOSERS"))

	for _, diagnoser := range enabledDiagnoserNames {
		diagnosers = append(diagnosers, allDiagnosersByName[diagnoser])
	}

	return diagnosers
}

// selectedExporters select the exporters to run
func selectExporters(allExporters map[string]interfaces.Exporter) []interfaces.Exporter {
	exporters := []interfaces.Exporter{}

	//read list of collectors that are enabled
	enabledExporterNames := strings.Fields(os.Getenv("ENABLED_EXPORTERS"))

	for _, exporter := range enabledExporterNames {
		exporters = append(exporters, allExporters[exporter])
	}

	return exporters
}

// runCollectors run the collectors
func runCollectors(collectors []interfaces.Collector, waitgroup *sync.WaitGroup) {
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
			err = c.Export()
			if err != nil {
				log.Printf("Collector: %s, export data failed: %v", c.GetName(), err)
			}
		}(c)
	}
}

// runDiagnosers run the diagnosers
func runDiagnosers(diagnosers []interfaces.Diagnoser, diagnoserGrp *sync.WaitGroup) {
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
			err = d.Export()
			if err != nil {
				log.Printf("Diagnoser: %s, export data failed: %v", d.GetName(), err)
			}
		}(d)
	}
}

// runExporters run the exporters
func runExporters(exporters []interfaces.Exporter, filesToExport []string) error {
	var result error
	for _, exporter := range exporters {
		if err := exporter.Export(filesToExport); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}

// zipAndExport zip the results
func zipOutputDirectory() (zipFiles []string, error error) {
	hostName, err := utils.GetHostName()
	if err != nil {
		return nil, err
	}

	creationTimeStamp, err := utils.GetCreationTimeStamp()
	if err != nil {
		return nil, err
	}

	sourcePathOnHost := "/var/log/aks-periscope/" + strings.Replace(creationTimeStamp, ":", "-", -1) + "/" + hostName
	zipFileOnHost := sourcePathOnHost + "/" + hostName + ".zip"
	zipFileOnContainer := strings.TrimPrefix(zipFileOnHost, "/var/log")

	_, err = utils.RunCommandOnHost("zip", "-r", zipFileOnHost, sourcePathOnHost)
	if err != nil {
		return nil, err
	}

	return []string{zipFileOnContainer}, nil
}

func contains(flagsList []string, flag string) bool {
	for _, f := range flagsList {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}
