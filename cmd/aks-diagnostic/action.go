package main

import (
	"github.com/Azure/aks-diagnostic-tool/pkg/collector"
	"github.com/Azure/aks-diagnostic-tool/pkg/processor"
)

// Action defines a diagnostic action
type Action struct {
	Name      string
	Collector func(string) ([]string, error)
	Processor func(string, []string) error
}

// NetworkConnectivityAction defines an action to collect and process network connectivity metrics
var NetworkConnectivityAction = Action{
	Name:      "networkconnectivity",
	Collector: collector.CollectNetworkConnectivity,
	Processor: processor.ProcessNetworkConnectivity,
}

// ContainerLogsAction defines an action to collect and process container logs
var ContainerLogsAction = Action{
	Name:      "containerlogs",
	Collector: collector.CollectContainerLogs,
	Processor: nil,
}

// ServiceLogsAction defines an action to collect and process service logs
var ServiceLogsAction = Action{
	Name:      "servicelogs",
	Collector: collector.CollectServiceLogs,
	Processor: nil,
}

// IPTablesAction defines an action to collect and process iptables info
var IPTablesAction = Action{
	Name:      "iptables",
	Collector: collector.CollectIPTables,
	Processor: nil,
}

// ProvisionLogsAction defines an action to collect and process provision logs
var ProvisionLogsAction = Action{
	Name:      "provisionlogs",
	Collector: collector.CollectProvisionLogs,
	Processor: nil,
}

// DNSAction defines an action to collect and process dns info
var DNSAction = Action{
	Name:      "dns",
	Collector: collector.CollectDNS,
	Processor: processor.ProcessDNS,
}
