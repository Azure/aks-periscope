package main

import (
	"os"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/datacollector"
	"github.com/Azure/aks-diagnostic-tool/pkg/dataexporter"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

func main() {

	cluster := os.Getenv("CLUSTER")
	if cluster == "" {
		cluster = "default"
	}

	containerLogs, _ := datacollector.PollContainerLogs("kube-system")
	dataexporter.WriteToBlob(cluster, "containerlogs", containerLogs)

	systemlogs, _ := datacollector.PollSystemLogs([]string{"docker", "kubelet"})
	dataexporter.WriteToBlob(cluster, "systemlogs", systemlogs)

	connectionsToCheck := []string{"google.com:80", "azurecr.io:80", "mcr.microsoft.com:80", "kubernetes.default.svc.cluster.local:443"}
	fqdn, err := utils.GetFQDN()

	if err == nil && fqdn != "" {
		connectionsToCheck = append(connectionsToCheck, fqdn+":443")
		connectionsToCheck = append(connectionsToCheck, fqdn+":9000")
	}

	networkConnectivity, _ := datacollector.CheckNetworkConnectivity(connectionsToCheck)
	dataexporter.WriteToBlob(cluster, "networkconnectivity", []string{networkConnectivity})

	iptables, _ := datacollector.DumpIPTables()
	dataexporter.WriteToBlob(cluster, "iptables", []string{iptables})

	snapshot, _ := datacollector.Snapshot()
	dataexporter.WriteToBlob(cluster, "snapshot", []string{snapshot})

	provisionLogs, _ := datacollector.ProvisionLogs()
	dataexporter.WriteToBlob(cluster, "provision", []string{provisionLogs})

	time.Sleep(24 * time.Hour)
}
