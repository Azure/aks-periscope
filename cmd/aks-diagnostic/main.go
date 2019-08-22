package main

import "github.com/Azure/aks-diagnostic-tool/pkg/exporter"

func main() {
	actions := []Action{}
	actions = append(actions, ContainerLogsAction)
	actions = append(actions, ServiceLogsAction)
	actions = append(actions, NetworkConnectivityAction)
	actions = append(actions, IPTablesAction)
	actions = append(actions, ProvisionLogsAction)
	actions = append(actions, DNSAction)

	for _, action := range actions {
		go func(action Action) {
			files, _ := action.Collector(action.Name)
			if action.Processor != nil {
				action.Processor(action.Name, files)
			}
		}(action)
	}

	exporter.ExportToAZBlob()

	select {}
}
