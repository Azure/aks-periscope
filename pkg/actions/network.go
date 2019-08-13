package actions

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// CheckNetworkConnectivity check network connectivity
func CheckNetworkConnectivity(urls []string) (string, error) {
	rootPath := filepath.Join("/aks-diagnostic", utils.GetHostName())
	err := os.MkdirAll(rootPath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	networkConnectivityFile := filepath.Join(rootPath, "networkconnectivity")
	file, _ := os.Create(networkConnectivityFile)
	defer file.Close()

	fmt.Fprintf(file, "%50v%20v%100v\n", "URL", "Connectivity", "Error")
	timeout := time.Duration(60 * time.Second)

	for _, url := range urls {
		_, err := net.DialTimeout("tcp", url, timeout)
		if err != nil {
			log.Println("Site unreachable, error: ", err)
		}
		fmt.Fprintf(file, "%50v%20v%100v\n", url, err == nil, err)
	}

	return networkConnectivityFile, nil
}
