package main

import (
	"fmt"

	"github.com/Azure/aks-diagnostic-tool/pkg/action"
	"github.com/Azure/aks-diagnostic-tool/pkg/exporter"
	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
)

func main() {
	exporter := &exporter.AzureBlobExporter{}

	actions := []interfaces.Action{}
	actions = append(actions, action.NewContainerLogsAction(5, 30, 60, exporter))
	actions = append(actions, action.NewSystemLogsAction(5, 30, 60, exporter))
	actions = append(actions, action.NewNetworkOutboundAction(5, 30, 60, exporter))
	actions = append(actions, action.NewIPTablesAction(5, 30, 60, exporter))
	actions = append(actions, action.NewProvisionLogsAction(5, 30, 60, exporter))
	actions = append(actions, action.NewDNSAction(5, 30, 60, exporter))
	actions = append(actions, action.NewKubeObjectsAction(5, 30, 60, exporter))
	actions = append(actions, action.NewKubeletCmdAction(5, 30, 60, exporter))

	for _, a := range actions {
		go func(a interfaces.Action) {
			collectFiles, err := a.Collect()
			if err != nil {
				fmt.Printf("Error in collect for %s: %+v", a.GetName(), err)
				return
			}

			processFiles, err := a.Process(collectFiles)
			if err != nil {
				fmt.Printf("Error in process for %s: %+v", a.GetName(), err)
				return
			}

			err = a.Export(exporter, collectFiles, processFiles)
			if err != nil {
				fmt.Printf("Error in export for %s: %+v", a.GetName(), err)
				return
			}
		}(a)
	}

	select {}
}
