package main

import (
	"fmt"
	"time"

	"github.com/Azure/aks-periscope/pkg/action"
	"github.com/Azure/aks-periscope/pkg/exporter"
	"github.com/Azure/aks-periscope/pkg/interfaces"
)

func main() {
	exporter := &exporter.AzureBlobExporter{}

	actions := []interfaces.Action{}
	actions = append(actions, action.NewContainerLogsAction(60, 5, 10, exporter))
	actions = append(actions, action.NewSystemLogsAction(60, 5, 10, exporter))
	actions = append(actions, action.NewNetworkOutboundAction(5, 5, 10, exporter))
	actions = append(actions, action.NewIPTablesAction(300, 5, 10, exporter))
	actions = append(actions, action.NewProvisionLogsAction(300, 5, 10, exporter))
	actions = append(actions, action.NewDNSAction(300, 5, 10, exporter))
	actions = append(actions, action.NewKubeObjectsAction(60, 5, 10, exporter))
	actions = append(actions, action.NewKubeletCmdAction(300, 5, 10, exporter))

	for _, a := range actions {
		go func(a interfaces.Action) {
			iTick := 0
			isRunning := false
			ticker := time.NewTicker(time.Duration(a.GetCollectIntervalInSeconds()) * time.Second)
			for ; true; <-ticker.C {
				if !isRunning {
					isRunning = true

					fmt.Printf("%s Collect %20v, iteration: %d\n", time.Now().Format(time.RFC3339), a.GetName(), iTick)
					err := a.Collect()
					if err != nil {
						fmt.Printf("Collect %s failed at iteration: %d: %+v\n", a.GetName(), iTick, err)
					}

					if iTick%a.GetCollectCountForProcess() == 0 {
						fmt.Printf("%s Process %20v, iteration: %d\n", time.Now().Format(time.RFC3339), a.GetName(), iTick/a.GetCollectCountForProcess())
						err := a.Process()
						if err != nil {
							fmt.Printf("Process %s failed at iteration: %d: %+v\n", a.GetName(), iTick/a.GetCollectCountForProcess(), err)
						}
					}

					if iTick%a.GetCollectCountForExport() == 0 {
						fmt.Printf("%s Export %21v, iteration: %d\n", time.Now().Format(time.RFC3339), a.GetName(), iTick/a.GetCollectCountForExport())
						err := a.Export()
						if err != nil {
							fmt.Printf("Export %s failed at iteration: %d: %+v", a.GetName(), iTick/a.GetCollectCountForExport(), err)
						}
					}

					iTick++
					isRunning = false
				}
			}
		}(a)
	}

	select {}
}
