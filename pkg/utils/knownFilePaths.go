package utils

import (
	"fmt"
)

type KnownFilePaths struct {
	AzureJson               string
	AzureStackCloudJson     string
	ResolvConfHost          string
	ResolvConfContainer     string
	AzureStackCertHost      string
	AzureStackCertContainer string
}

// GetKnownFilePaths get known file paths
func GetKnownFilePaths(runtimeInfo *RuntimeInfo) (*KnownFilePaths, error) {
	os := runtimeInfo.OSIdentifier
	switch os {
	case "windows":
		return &KnownFilePaths{
			AzureJson:           "/k/azure.json",
			AzureStackCloudJson: "/k/azurestackcloud.json",
		}, nil
	case "linux":
		// Since Azure Stack Hub does not support multiple node pools, we assume we don't need to worry about this for Windows
		// https://docs.microsoft.com/en-us/azure-stack/user/aks-overview?view=azs-2108#supported-platform-features
		return &KnownFilePaths{
			AzureJson:               "/etc/kubernetes/azure.json",
			AzureStackCloudJson:     "/etc/kubernetes/azurestackcloud.json",
			ResolvConfHost:          "/etchostlogs/resolv.conf",
			ResolvConfContainer:     "/etc/resolv.conf",
			AzureStackCertHost:      "/etchostlogs/ssl/certs/azsCertificate.pem",
			AzureStackCertContainer: "/etc/ssl/certs/azsCertificate.pem",
		}, nil
	default:
		return nil, fmt.Errorf("unexpected OS: %s", os)
	}
}
