package main

import (
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/actions"
	"github.com/Azure/aks-diagnostic-tool/pkg/storage"
)

func main() {
	containerLogs, _ := actions.PollContainerLogs("kube-system")
	storage.WriteToBlob("containerlogs", containerLogs)

	systemlogs, _ := actions.PollSystemLogs([]string{"docker", "kubelet"})
	storage.WriteToBlob("systemlogs", systemlogs)

	networkConnectivity, _ := actions.CheckNetworkConnectivity([]string{"google.com:80", "azurecr.io:80", "bad.site:80"})
	storage.WriteToBlob("networkconnectivity", []string{networkConnectivity})

	time.Sleep(24 * time.Hour)
}
