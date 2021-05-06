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
)

func main() {
	zipAndExportMode := true
	exporter := &exporter.AzureBlobExporter{}
	var waitgroup sync.WaitGroup

	err := utils.CreateCRD()
	if err != nil {
		log.Printf("Failed to create CRD: %+v", err)
	}

	//create map of all possible collectors by name, discrete vars for each collector as some are diagnoser dependecies
	allCollectorsByName := make(map[string]interfaces.Collector)
	containerLogsCollector := collector.NewContainerLogsCollector(exporter)
	allCollectorsByName["containerlogs"] = containerLogsCollector
	systemLogsCollector := collector.NewSystemLogsCollector(exporter)
	allCollectorsByName["systemlogs"] = systemLogsCollector
	networkOutboundCollector := collector.NewNetworkOutboundCollector(5, exporter)
	allCollectorsByName["networkoutbound"] = networkOutboundCollector
	ipTablesCollector := collector.NewIPTablesCollector(exporter)
	allCollectorsByName["iptables"] = ipTablesCollector
	nodeLogsCollector := collector.NewNodeLogsCollector(exporter)
	allCollectorsByName["nodelogs"] = nodeLogsCollector
	dnsCollector := collector.NewDNSCollector(exporter)
	allCollectorsByName["dns"] = dnsCollector
	kubeObjectsCollector := collector.NewKubeObjectsCollector(exporter)
	allCollectorsByName["kubeobjects"] = kubeObjectsCollector
	kubeletCmdCollector := collector.NewKubeletCmdCollector(exporter)
	allCollectorsByName["kubeletcmd"] = kubeletCmdCollector
	systemPerfCollector := collector.NewSystemPerfCollector(exporter)
	allCollectorsByName["systemperf"] = systemPerfCollector

	//read list of collectors that are enabled
	enabledCollectorNames := strings.Fields(os.Getenv("ENABLED_COLLECTORS"))

	//gather those collectors which are enabled by selecting from allCollectorsByName
	collectors := []interfaces.Collector{}
	for _, collector := range enabledCollectorNames {
		collectors = append(collectors, allCollectorsByName[collector])
	}

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

	waitgroup.Wait()

	//create map of all possible diagnosers by name
	allDiagnosersByName := make(map[string]interfaces.Diagnoser)
	allDiagnosersByName["networkconfig"] = diagnoser.NewNetworkConfigDiagnoser(dnsCollector, kubeletCmdCollector, exporter)
	allDiagnosersByName["networkoutbound"] = diagnoser.NewNetworkOutboundDiagnoser(networkOutboundCollector, exporter)

	//read list of diagnosers that are enabled
	enabledDiagnoserNames := strings.Fields(os.Getenv("ENABLED_DIAGNOSERS"))

	//gather those diagnosers which are enabled by selecting from allDiagnosersByName
	diagnosers := []interfaces.Diagnoser{}
	for _, diagnoser := range enabledDiagnoserNames {
		diagnosers = append(diagnosers, allDiagnosersByName[diagnoser])
	}

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

	waitgroup.Wait()

	if zipAndExportMode {
		log.Print("Zip and export result files")
		err := zipAndExport(exporter)
		if err != nil {
			log.Printf("Failed to zip and export result files: %+v", err)
		}
	}

	select {}
}

// zipAndExport zip the results and export
func zipAndExport(exporter interfaces.Exporter) error {
	hostName, err := utils.GetHostName()
	if err != nil {
		return err
	}

	creationTimeStamp, err := utils.GetCreationTimeStamp()
	if err != nil {
		return err
	}

	sourcePathOnHost := "/var/log/aks-periscope/" + strings.Replace(creationTimeStamp, ":", "-", -1) + "/" + hostName
	zipFileOnHost := sourcePathOnHost + "/" + hostName + ".zip"
	zipFileOnContainer := strings.TrimPrefix(zipFileOnHost, "/var/log")

	_, err = utils.RunCommandOnHost("zip", "-r", zipFileOnHost, sourcePathOnHost)
	if err != nil {
		return err
	}

	err = exporter.Export([]string{zipFileOnContainer})
	if err != nil {
		return err
	}

	return nil
}
