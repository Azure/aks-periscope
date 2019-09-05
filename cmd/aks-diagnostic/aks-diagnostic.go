package main

import (
	"fmt"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/action"
	"github.com/Azure/aks-diagnostic-tool/pkg/exporter"
	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
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

					err := a.Collect()
					if err != nil {
						fmt.Printf("Error in collect for %s: %+v\n", a.GetName(), err)
						return
					}

					if iTick%a.GetCollectCountForProcess() == 0 {
						err := a.Process()
						if err != nil {
							fmt.Printf("Error in collect for %s: %+v\n", a.GetName(), err)
							return
						}
					}

					if iTick%a.GetCollectCountForExport() == 0 {
						err := a.Export()
						if err != nil {
							fmt.Printf("Error in export for %s: %+v\n", a.GetName(), err)
							return
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
