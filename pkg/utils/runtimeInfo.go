package utils

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
)

type Feature string

const (
	WindowsHpc Feature = "WINHPC"
)

func getKnownFeatures() []Feature {
	return []Feature{WindowsHpc}
}

type RuntimeInfo struct {
	RunId                   string
	OSIdentifier            string
	HostNodeName            string
	CollectorList           []string
	KubernetesObjects       []string
	NodeLogs                []string
	ContainerLogsNamespaces []string
	StorageAccountName      string
	StorageSasKey           string
	StorageContainerName    string
	StorageSasKeyType       string
	Features                map[Feature]bool
}

// GetRuntimeInfo gets runtime info
func GetRuntimeInfo() (*RuntimeInfo, error) {
	runId := os.Getenv("DIAGNOSTIC_RUN_ID")

	osIdentifier := runtime.GOOS

	// We can't use `os.Hostname` for this, because this gives us the _container_ hostname (i.e. the pod name, by default).
	// An earlier approach was to `cat /etc/hostname` but that will not work for Windows containers.
	// Instead we expect the host node name to be exposed to the pod in an environment variable, via the 'downward API', see:
	// https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/#use-pod-fields-as-values-for-environment-variables
	hostName := os.Getenv("HOST_NODE_NAME")
	if len(hostName) == 0 {
		return nil, errors.New("variable HOST_NODE_NAME value not set for container")
	}

	collectorList := strings.Fields(os.Getenv("COLLECTOR_LIST"))
	kubernetesObjects := strings.Fields(os.Getenv("DIAGNOSTIC_KUBEOBJECTS_LIST"))
	var nodeLogs []string
	if osIdentifier == "linux" {
		nodeLogs = strings.Fields(os.Getenv("DIAGNOSTIC_NODELOGS_LIST_LINUX"))
	} else {
		nodeLogs = strings.Fields(os.Getenv("DIAGNOSTIC_NODELOGS_LIST_WINDOWS"))
	}
	containerLogsNamespaces := strings.Fields(os.Getenv("DIAGNOSTIC_CONTAINERLOGS_LIST"))

	storageAccountName := os.Getenv("AZURE_BLOB_ACCOUNT_NAME")
	storageSasKey := os.Getenv("AZURE_BLOB_SAS_KEY")
	storageContainerName := os.Getenv("AZURE_BLOB_CONTAINER_NAME")
	storageSasKeyType := os.Getenv("AZURE_STORAGE_SAS_KEY_TYPE")

	features := map[Feature]bool{}
	for _, feature := range getKnownFeatures() {
		enabled := os.Getenv(fmt.Sprintf("FEATURE_%s", feature))
		if len(enabled) > 0 {
			features[feature] = true
		}
	}

	return &RuntimeInfo{
		RunId:                   runId,
		OSIdentifier:            osIdentifier,
		HostNodeName:            hostName,
		CollectorList:           collectorList,
		KubernetesObjects:       kubernetesObjects,
		NodeLogs:                nodeLogs,
		ContainerLogsNamespaces: containerLogsNamespaces,
		StorageAccountName:      storageAccountName,
		StorageSasKey:           storageSasKey,
		StorageContainerName:    storageContainerName,
		StorageSasKeyType:       storageSasKeyType,
		Features:                features,
	}, nil
}

func (runtimeInfo *RuntimeInfo) HasFeature(feature Feature) bool {
	_, ok := runtimeInfo.Features[feature]
	return ok
}
