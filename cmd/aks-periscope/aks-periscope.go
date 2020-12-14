package main

import (
	"log"
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

	err := utils.CreateCRD()
	if err != nil {
		log.Fatalf("Failed to create CRD: %+v", err)
	}

	exporter, err := exporter.NewAzureBlobExporter()
	if err != nil {
		log.Fatal(err)
	}

	var waitgroup sync.WaitGroup

	collectors := []interfaces.Collector{}
	containerLogsCollector := collector.NewContainerLogsCollector(exporter)
	collectors = append(collectors, containerLogsCollector)
	systemLogsCollector := collector.NewSystemLogsCollector(exporter)
	collectors = append(collectors, systemLogsCollector)
	networkOutboundCollector := collector.NewNetworkOutboundCollector(5, exporter)
	collectors = append(collectors, networkOutboundCollector)
	ipTablesCollector := collector.NewIPTablesCollector(exporter)
	collectors = append(collectors, ipTablesCollector)
	nodeLogsCollector := collector.NewNodeLogsCollector(exporter)
	collectors = append(collectors, nodeLogsCollector)
	dnsCollector := collector.NewDNSCollector(exporter)
	collectors = append(collectors, dnsCollector)
	kubeObjectsCollector := collector.NewKubeObjectsCollector(exporter)
	collectors = append(collectors, kubeObjectsCollector)
	kubeletCmdCollector := collector.NewKubeletCmdCollector(exporter)
	collectors = append(collectors, kubeletCmdCollector)
	systemPerfCollector := collector.NewSystemPerfCollector(exporter)
	collectors = append(collectors, systemPerfCollector)

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

	diagnosers := []interfaces.Diagnoser{}
	diagnosers = append(diagnosers, diagnoser.NewNetworkConfigDiagnoser(dnsCollector, kubeletCmdCollector, exporter))
	diagnosers = append(diagnosers, diagnoser.NewNetworkOutboundDiagnoser(networkOutboundCollector, exporter))

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
