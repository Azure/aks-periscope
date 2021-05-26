package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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

	clusterType := os.Getenv("CLUSTER_TYPE")

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

	if strings.EqualFold(clusterType, "connectedCluster") {
		collectors = append(collectors, containerLogsCollector)
		collectors = append(collectors, dnsCollector)
		collectors = append(collectors, helmCollector)
		collectors = append(collectors, kubeObjectsCollector)
		collectors = append(collectors, networkOutboundCollector)

	} else {
		collectors = append(collectors, containerLogsCollector)
		collectors = append(collectors, dnsCollector)
		collectors = append(collectors, kubeObjectsCollector)
		collectors = append(collectors, networkOutboundCollector)
		collectors = append(collectors, systemLogsCollector)
		collectors = append(collectors, ipTablesCollector)
		collectors = append(collectors, nodeLogsCollector)
		collectors = append(collectors, kubeletCmdCollector)
		collectors = append(collectors, systemPerfCollector)
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

	sourcePathOnHost := fmt.Sprintf("/var/log/aks-periscope/%s/%s", strings.Replace(creationTimeStamp, ":", "-", -1), hostName)
	zipFileOnHost := fmt.Sprintf("%s/%s.zip", sourcePathOnHost, hostName)
	zipFileOnContainer := strings.TrimPrefix(zipFileOnHost, "/var/log")

	if err = createZip(zipFileOnHost, sourcePathOnHost); err != nil {
		return fmt.Errorf("create zip: %w", err)
	}

	return exporter.Export([]string{zipFileOnContainer})
}

func createZip(sourcePath, destinationPath string) error {
	destinationFile, err := os.Create(sourcePath)
	if err != nil {
		return fmt.Errorf("file %s cannot be created: %w", sourcePath, err)
	}

	w := zip.NewWriter(destinationFile)
	defer w.Close()

	return filepath.Walk(sourcePath, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(filePath, filepath.Dir(sourcePath))
		zipFile, err := w.Create(relPath)
		if err != nil {
			return err
		}

		fsFile, err := os.Open(filePath)
		if err != nil {
			return err
		}

		_, err = io.Copy(zipFile, fsFile)

		return err
	})
}
