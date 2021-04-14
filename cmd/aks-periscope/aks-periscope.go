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

	collectors := []interfaces.Collector{}
	containerLogsCollector := collector.NewContainerLogsCollector(exporter)
	networkOutboundCollector := collector.NewNetworkOutboundCollector(5, exporter)
	dnsCollector := collector.NewDNSCollector(exporter)
	kubeObjectsCollector := collector.NewKubeObjectsCollector(exporter)
	systemLogsCollector := collector.NewSystemLogsCollector(exporter)
	ipTablesCollector := collector.NewIPTablesCollector(exporter)
	nodeLogsCollector := collector.NewNodeLogsCollector(exporter)
	kubeletCmdCollector := collector.NewKubeletCmdCollector(exporter)
	systemPerfCollector := collector.NewSystemPerfCollector(exporter)
	helmCollector := collector.NewHelmCollector(exporter)
	osmCollector := collector.NewOsmCollector(exporter)

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

	collectorGrp := new(sync.WaitGroup)

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

	collectorGrp.Wait()

	diagnosers := []interfaces.Diagnoser{}
	diagnosers = append(diagnosers, diagnoser.NewNetworkConfigDiagnoser(dnsCollector, kubeletCmdCollector, exporter))
	diagnosers = append(diagnosers, diagnoser.NewNetworkOutboundDiagnoser(networkOutboundCollector, exporter))

	diagnoserGrp := new(sync.WaitGroup)

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

	diagnoserGrp.Wait()

	if zipAndExportMode {
		log.Print("Zip and export result files")
		err := zipAndExport(exporter)
		if err != nil {
			log.Fatalf("Failed to zip and export result files: %v", err)
		}
	}
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

	// TODO: Hack: for now AKS-Periscope is running as a deamonset so it shall not stop (or the pod will be restarted)
	// Revert from https://github.com/Azure/aks-periscope/blob/b98d66a238e942158ef2628a9315b58937ff9c8f/cmd/aks-periscope/aks-periscope.go#L70
	select {}

	// TODO: remove this //nolint comment once the select{} has been removed
	//nolint:govet
	return nil
}

func contains(flagsList []string, flag string) bool {
	for _, f := range flagsList {
		if strings.EqualFold(f, flag) {
			return true
		}
	}
	return false
}
