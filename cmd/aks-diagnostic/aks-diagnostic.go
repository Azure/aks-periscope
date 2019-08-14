package main

import (
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/actions"
	"github.com/Azure/aks-diagnostic-tool/pkg/storage"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

func main() {
	timeStamp := time.Now().Format("200601021504")

	containerLogs, _ := actions.PollContainerLogs("kube-system")
	storage.WriteToBlob("containerlogs-"+timeStamp, containerLogs)

	systemlogs, _ := actions.PollSystemLogs([]string{"docker", "kubelet"})
	storage.WriteToBlob("systemlogs-"+timeStamp, systemlogs)

	connectionsToCheck := []string{"google.com:80", "azurecr.io:80", "mcr.microsoft.com:80", "kubernetes.default.svc.cluster.local:443"}
	fqdn, err := utils.GetFQDN()

	if err == nil && fqdn != "" {
		connectionsToCheck = append(connectionsToCheck, fqdn+":443")
		connectionsToCheck = append(connectionsToCheck, fqdn+":9000")
	}

	networkConnectivity, _ := actions.CheckNetworkConnectivity(connectionsToCheck)
	storage.WriteToBlob("networkconnectivity-"+timeStamp, []string{networkConnectivity})

	iptables, _ := actions.DumpIPTables()
	storage.WriteToBlob("iptables-"+timeStamp, []string{iptables})

	snapshot, _ := actions.Snapshot()
	storage.WriteToBlob("snapshot-"+timeStamp, []string{snapshot})

	time.Sleep(24 * time.Hour)
}
