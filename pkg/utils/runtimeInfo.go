package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/hashicorp/go-multierror"
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
func GetRuntimeInfo(fs interfaces.FileSystemAccessor, filePaths *KnownFilePaths) (*RuntimeInfo, error) {
	var errs error

	// Config
	runId, errs := readFileContent(fs, filePaths.GetConfigPath(RunIdKey), true, errs)
	collectorList, errs := readFileContent(fs, filePaths.GetConfigPath(CollectorListKey), false, errs)
	kubernetesObjects, errs := readFileContent(fs, filePaths.GetConfigPath(KubeObjectsListKey), false, errs)
	nodeLogs, errs := readFileContent(fs, filePaths.NodeLogsList, false, errs)
	containerLogsNamespaces, errs := readFileContent(fs, filePaths.GetConfigPath(ContainerLogsListKey), false, errs)

	// Secret
	storageAccountName, errs := readFileContent(fs, filePaths.GetSecretPath(AccountNameKey), false, errs)
	storageSasKey, errs := readFileContent(fs, filePaths.GetSecretPath(SasTokenKey), false, errs)
	storageContainerName, errs := readFileContent(fs, filePaths.GetSecretPath(ContainerNameKey), false, errs)
	storageSasKeyType, errs := readFileContent(fs, filePaths.GetSecretPath(SasTokenTypeKey), false, errs)

	// We can't use `os.Hostname` for this, because this gives us the _container_ hostname (i.e. the pod name, by default).
	// An earlier approach was to `cat /etc/hostname` but that will not work for Windows containers.
	// Instead we expect the host node name to be exposed to the pod in an environment variable, via the 'downward API', see:
	// https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/#use-pod-fields-as-values-for-environment-variables
	hostName := os.Getenv("HOST_NODE_NAME")
	if len(hostName) == 0 {
		errs = multierror.Append(errs, errors.New("variable HOST_NODE_NAME value not set for container"))
	}

	features := map[Feature]bool{}
	for _, feature := range getKnownFeatures() {
		featureFilePath := filePaths.GetFeaturePath(feature)
		var enabled string
		enabled, errs = readFileContent(fs, featureFilePath, false, errs)
		if len(enabled) > 0 {
			features[feature] = true
		}
	}

	if errs != nil {
		return nil, errs
	}

	return &RuntimeInfo{
		RunId:                   runId,
		HostNodeName:            hostName,
		CollectorList:           strings.Fields(collectorList),
		KubernetesObjects:       strings.Fields(kubernetesObjects),
		NodeLogs:                strings.Fields(nodeLogs),
		ContainerLogsNamespaces: strings.Fields(containerLogsNamespaces),
		StorageAccountName:      storageAccountName,
		StorageSasKey:           storageSasKey,
		StorageContainerName:    storageContainerName,
		StorageSasKeyType:       storageSasKeyType,
		Features:                features,
	}, nil
}

func readFileContent(fs interfaces.FileSystemAccessor, filePath string, mandatory bool, readErrors error) (string, error) {
	value, err := GetFileContent(fs, filePath)
	if err != nil {
		return "", multierror.Append(readErrors, fmt.Errorf("error reading %s: %w", filePath, err))
	}
	if mandatory && len(value) == 0 {
		return "", multierror.Append(readErrors, fmt.Errorf("mandatory file has no content: %s", filePath))
	}
	return value, readErrors
}

func (runtimeInfo *RuntimeInfo) HasFeature(feature Feature) bool {
	_, ok := runtimeInfo.Features[feature]
	return ok
}
