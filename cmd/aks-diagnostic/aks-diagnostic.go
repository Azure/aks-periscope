package main

import (
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/actions"
	"github.com/Azure/aks-diagnostic-tool/pkg/storage"
)

func main() {

	timeStamp := time.Now().Format("20060102150405")

	containerLogs, _ := actions.PollContainerLogs("kube-system")
	storage.WriteToBlob("containerlogs-"+timeStamp, containerLogs)

	systemlogs, _ := actions.PollSystemLogs([]string{"docker", "kubelet"})
	storage.WriteToBlob("systemlogs-"+timeStamp, systemlogs)

	networkConnectivity, _ := actions.CheckNetworkConnectivity([]string{"google.com:80", "azurecr.io:80", "bad.site:80"})
	storage.WriteToBlob("networkconnectivity-"+timeStamp, []string{networkConnectivity})

	iptables, _ := actions.DumpIPTables()
	storage.WriteToBlob("iptables-"+timeStamp, []string{iptables})

	time.Sleep(24 * time.Hour)
}
