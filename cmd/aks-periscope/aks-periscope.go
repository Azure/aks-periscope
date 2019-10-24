package main

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Azure/aks-periscope/pkg/action"
	"github.com/Azure/aks-periscope/pkg/exporter"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

func main() {
	runInContinuousMode := false
	zipAndExportMode := true

	exporter := &exporter.AzureBlobExporter{}

	actions := []interfaces.Action{}
	actions = append(actions, action.NewContainerLogsAction(60, 5, 10, exporter))
	actions = append(actions, action.NewSystemLogsAction(60, 5, 10, exporter))
	actions = append(actions, action.NewNetworkOutboundAction(5, 5, 10, exporter))
	actions = append(actions, action.NewIPTablesAction(300, 5, 10, exporter))
	actions = append(actions, action.NewOnDemandLogsAction(300, 5, 10, exporter))
	actions = append(actions, action.NewDNSAction(300, 5, 10, exporter))
	actions = append(actions, action.NewKubeObjectsAction(60, 5, 10, exporter))
	actions = append(actions, action.NewKubeletCmdAction(300, 5, 10, exporter))
	actions = append(actions, action.NewSystemPerfAction(300, 5, 10, exporter))

	var waitgroup sync.WaitGroup

	for _, a := range actions {
		waitgroup.Add(1)
		go func(a interfaces.Action) {
			iTick := 0
			isRunning := false
			ticker := time.NewTicker(time.Duration(a.GetCollectIntervalInSeconds()) * time.Second)
			for ; true; <-ticker.C {
				if !isRunning {
					isRunning = true

					log.Printf("Action: %s, collect data, iteration: %d\n", a.GetName(), iTick)
					err := a.Collect()
					if err != nil {
						log.Printf("Action: %s, collect data failed at iteration: %d: %+v\n", a.GetName(), iTick, err)
					}

					if iTick%a.GetCollectCountForProcess() == 0 {
						log.Printf("Action: %s, process data, iteration: %d\n", a.GetName(), iTick/a.GetCollectCountForProcess())
						err := a.Process()
						if err != nil {
							log.Printf("Action: %s, process data failed at iteration: %d: %+v\n", a.GetName(), iTick/a.GetCollectCountForProcess(), err)
						}
					}

					if iTick%a.GetCollectCountForExport() == 0 {
						log.Printf("Action: %s, export data, iteration: %d\n", a.GetName(), iTick/a.GetCollectCountForExport())
						err := a.Export()
						if err != nil {
							log.Printf("Action: %s, export data failed at iteration: %d: %+v", a.GetName(), iTick/a.GetCollectCountForExport(), err)
						}
					}

					iTick++
					isRunning = false
				}

				if !runInContinuousMode {
					break
				}
			}

			waitgroup.Done()
		}(a)
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
