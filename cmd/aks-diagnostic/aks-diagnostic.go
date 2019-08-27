package main

import (
	"github.com/Azure/aks-diagnostic-tool/pkg/action"
	"github.com/Azure/aks-diagnostic-tool/pkg/exporter"
	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
)

func main() {
	actions := []interfaces.Action{}
	actions = append(actions, &action.ContainerLogsAction{})
	actions = append(actions, &action.ServiceLogsAction{})
	actions = append(actions, &action.NetworkConnectivityAction{})
	actions = append(actions, &action.IPTablesAction{})
	actions = append(actions, &action.ProvisionLogsAction{})
	actions = append(actions, &action.DNSAction{})
	actions = append(actions, &action.KubeObjectsAction{})
	actions = append(actions, &action.KubeletCmdAction{})

	for _, a := range actions {
		go func(a interfaces.Action) {
			files, _ := a.Collect()
			a.Process(files)
		}(a)
	}

	azureBlobExporter := exporter.AzureBlobExporter{
		IntervalInSeconds: 60,
	}

	azureBlobExporter.Export()

	select {}
}
