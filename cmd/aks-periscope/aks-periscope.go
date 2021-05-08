package main

import (
	"github.com/hashicorp/go-multierror"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/diagnoser"
	"github.com/Azure/aks-periscope/pkg/exporter"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

func main() {

	err := utils.CreateCRD()
	if err != nil {
		log.Printf("Failed to create CRD: %+v", err)
	}

	collectors, diagnosers, exporters := initializeComponents()

	var waitgroup sync.WaitGroup

	runCollectors(collectors, &waitgroup)
	waitgroup.Wait()

	runDiagnosers(diagnosers, &waitgroup)
	waitgroup.Wait()

	log.Print("Zip result files")
	outputs, err := zipOutputDirectory()
	if err != nil {
		log.Printf("Failed to zip result files: %+v", err)
	}

	log.Print("Run exporters for result files")
	err = runExporters(exporters, outputs)
	if err != nil {
		log.Printf("Failed to export result files: %+v", err)
	}

	select {}
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
		})

	//diagnosers
	//NOTE currently the collector instances are shared between the collector itself and things which use it as a dependency
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
		waitgroup.Add(1)
		go func(c interfaces.Collector) {
			log.Printf("Collector: %s, collect data\n", c.GetName())
			err := c.Collect()
			if err != nil {
				log.Printf("Collector: %s, collect data failed: %+v\n", c.GetName(), err)
			}

			log.Printf("Collector: %s, export data\n", c.GetName())
			err = c.Export()
			if err != nil {
				log.Printf("Collector: %s, export data failed: %+v\n", c.GetName(), err)
			}
			waitgroup.Done()
		}(c)
	}
}

// runDiagnosers run the diagnosers
func runDiagnosers(diagnosers []interfaces.Diagnoser, waitgroup *sync.WaitGroup) {
	for _, d := range diagnosers {
		waitgroup.Add(1)
		go func(d interfaces.Diagnoser) {
			log.Printf("Diagnoser: %s, diagnose data\n", d.GetName())
			err := d.Diagnose()
			if err != nil {
				log.Printf("Diagnoser: %s, diagnose data failed: %+v\n", d.GetName(), err)
			}

			log.Printf("Diagnoser: %s, export data\n", d.GetName())
			err = d.Export()
			if err != nil {
				log.Printf("Diagnoser: %s, export data failed: %+v\n", d.GetName(), err)
			}
			waitgroup.Done()
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
