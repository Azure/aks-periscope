package main

import (
	"os"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/actions"
	"github.com/Azure/aks-diagnostic-tool/pkg/storage"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

func main() {

	cluster := os.Getenv("CLUSTER")
	if cluster == "" {
		cluster = "default"
	}

	containerLogs, _ := actions.PollContainerLogs("kube-system")
	storage.WriteToBlob(cluster, "containerlogs", containerLogs)

	systemlogs, _ := actions.PollSystemLogs([]string{"docker", "kubelet"})
	storage.WriteToBlob(cluster, "systemlogs", systemlogs)

	connectionsToCheck := []string{"google.com:80", "azurecr.io:80", "mcr.microsoft.com:80", "kubernetes.default.svc.cluster.local:443"}
	fqdn, err := utils.GetFQDN()

	if err == nil && fqdn != "" {
		connectionsToCheck = append(connectionsToCheck, fqdn+":443")
		connectionsToCheck = append(connectionsToCheck, fqdn+":9000")
	}

	networkConnectivity, _ := actions.CheckNetworkConnectivity(connectionsToCheck)
	storage.WriteToBlob(cluster, "networkconnectivity", []string{networkConnectivity})

	iptables, _ := actions.DumpIPTables()
	storage.WriteToBlob(cluster, "iptables", []string{iptables})

	snapshot, _ := actions.Snapshot()
	storage.WriteToBlob(cluster, "snapshot", []string{snapshot})

	provisionLogs, _ := actions.ProvisionLogs()
	storage.WriteToBlob(cluster, "provision", []string{provisionLogs})

	time.Sleep(24 * time.Hour)
}
