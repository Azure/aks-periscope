package utils

import (
	"fmt"
	"path/filepath"
)

type KnownFilePaths struct {
	AzureJson               string
	AzureStackCloudJson     string
	WindowsLogsOutput       string
	ResolvConfHost          string
	ResolvConfContainer     string
	AzureStackCertHost      string
	AzureStackCertContainer string
	NodeLogsList            string
	Config                  string
	Secret                  string
}

type ConfigKey string
type SecretKey string

const (
	CollectorListKey     ConfigKey = "COLLECTOR_LIST"
	ContainerLogsListKey ConfigKey = "DIAGNOSTIC_CONTAINERLOGS_LIST"
	KubeObjectsListKey   ConfigKey = "DIAGNOSTIC_KUBEOBJECTS_LIST"
	NodeLogsLinuxKey     ConfigKey = "DIAGNOSTIC_NODELOGS_LIST_LINUX"
	NodeLogsWindowsKey   ConfigKey = "DIAGNOSTIC_NODELOGS_LIST_WINDOWS"
	RunIdKey             ConfigKey = "DIAGNOSTIC_RUN_ID"
)

const (
	AccountNameKey   SecretKey = "AZURE_BLOB_ACCOUNT_NAME"
	SasTokenKey      SecretKey = "AZURE_BLOB_SAS_KEY"
	ContainerNameKey SecretKey = "AZURE_BLOB_CONTAINER_NAME"
	SasTokenTypeKey  SecretKey = "AZURE_STORAGE_SAS_KEY_TYPE"
)

// GetKnownFilePaths get known file paths
func GetKnownFilePaths(osIdentifier OSIdentifier) (*KnownFilePaths, error) {
	switch osIdentifier {
	case Windows:
		return &KnownFilePaths{
			AzureJson:           "/k/azure.json",
			AzureStackCloudJson: "/k/azurestackcloud.json",
			WindowsLogsOutput:   "/k/periscope-diagnostic-output",
			NodeLogsList:        "/config/" + string(NodeLogsWindowsKey),
			Config:              "/config",
			Secret:              "/secret",
		}, nil
	case Linux:
		// Since Azure Stack Hub does not support multiple node pools, we assume we don't need to worry about this for Windows
		// https://docs.microsoft.com/en-us/azure-stack/user/aks-overview?view=azs-2108#supported-platform-features
		return &KnownFilePaths{
			AzureJson:               "/etc/kubernetes/azure.json",
			AzureStackCloudJson:     "/etc/kubernetes/azurestackcloud.json",
			ResolvConfHost:          "/etchostlogs/resolv.conf",
			ResolvConfContainer:     "/etc/resolv.conf",
			AzureStackCertHost:      "/etchostlogs/ssl/certs/azsCertificate.pem",
			AzureStackCertContainer: "/etc/ssl/certs/azsCertificate.pem",
			NodeLogsList:            "/config/" + string(NodeLogsLinuxKey),
			Config:                  "/config",
			Secret:                  "/secret",
		}, nil
	default:
		return nil, fmt.Errorf("unexpected OS: %s", osIdentifier)
	}
}

func (p *KnownFilePaths) GetConfigPath(key ConfigKey) string {
	return filepath.Join(p.Config, string(key))
}

func (p *KnownFilePaths) GetSecretPath(key SecretKey) string {
	return filepath.Join(p.Secret, string(key))
}

func (p *KnownFilePaths) GetFeaturePath(feature Feature) string {
	return filepath.Join(p.Config, fmt.Sprintf("FEATURE_%s", feature))
}
