package collector

import (
	"os"
	"path"
	"strings"

	"github.com/Azure/aks-periscope/pkg/utils"
)

// ContainerLogsCollectorContainerD defines a ContainerLogs Collector struct for containerd clusters
type ContainerLogsCollectorContainerD struct {
	data map[string]string
}

// NewContainerLogsCollectorContainerD is a constructor
func NewContainerLogsCollectorContainerD() *ContainerLogsCollectorContainerD {
	return &ContainerLogsCollectorContainerD{
		data: make(map[string]string),
	}
}

func (collector *ContainerLogsCollectorContainerD) GetName() string {
	return "containerlogscontainerd"
}

type ContainerLog struct {
	podname       string
	namespace     string
	containerName string
	containeruid  string
	filepath      string
}

type ContainerLogSelector struct {
	namespace           string
	containerNamePrefix string
}

//special selector which indicates logs from containers in all namespaces should be selected for collection
const allNamespacesSelector = "--all-namespaces"

// NOTE pod log files not currently used are in sub-directories of /var/log/pods and have format NAMESPACE_PODNAME_SOMEGUIDMEANINGTBD/CONTAINERNAME/#.log,
// where I assume # is incremented and a new log created for each container restart (starts at # == 0)
const containerLogDirectory = "/var/log/containers"

func (collector *ContainerLogsCollectorContainerD) GetData() map[string]string {
	return collector.data
}

// Collect implements the interface method
func (collector *ContainerLogsCollectorContainerD) Collect() error {
	selectorStrings := strings.Fields(os.Getenv("DIAGNOSTIC_CONTAINERLOGS_LIST"))

	containerLogSelectors := parseContainerLogSelectors(selectorStrings)

	allContainersEverRun, err := getAllContainerLogFilesThatHaveEverRunOnHost()
	if err != nil {
		return err
	}

	containerLogs := parseContainerLogFilenames(containerLogDirectory, allContainersEverRun)

	containerLogsToCollect := determineContainerLogsToCollect(containerLogs, containerLogSelectors)

	for _, containerLog := range containerLogsToCollect {
		output, err := utils.RunCommandOnHost("cat", containerLog.filepath)
		if err != nil {
			return err
		}

		collector.data[containerLog.containerName] = output
	}

	return nil
}

func determineContainerLogsToCollect(allContainers []ContainerLog, selectors []ContainerLogSelector) []ContainerLog {
	var selectedContainerLogs []ContainerLog
	for _, containerLog := range allContainers {
		for _, selector := range selectors {
			if doesSelectorSelectContainerLog(containerLog, selector) {
				selectedContainerLogs = append(selectedContainerLogs, containerLog)
			}
		}
	}
	return selectedContainerLogs
}

func doesSelectorSelectContainerLog(containerLog ContainerLog, selector ContainerLogSelector) bool {
	if selector.namespace == allNamespacesSelector {
		return true
	}

	return containerLog.namespace == selector.namespace && strings.HasPrefix(containerLog.containerName, selector.containerNamePrefix)
}

//TODO allow the raw struct objects to be defined directly in the deployment yaml and add additional required logic to DoesSelectorSelectContainerLog
func parseContainerLogSelectors(selectorStrings []string) []ContainerLogSelector {
	var containerLogSelectors []ContainerLogSelector

	for _, selectorString := range selectorStrings {
		selectorStringParts := strings.Split(selectorString, "/")

		if len(selectorStringParts) == 1 {
			containerLogSelectors = append(containerLogSelectors, ContainerLogSelector{
				namespace: selectorStringParts[0],
			})
		}
		if len(selectorStringParts) == 2 {
			containerLogSelectors = append(containerLogSelectors, ContainerLogSelector{
				namespace:           selectorStringParts[0],
				containerNamePrefix: selectorStringParts[1],
			})
		}
	}

	return containerLogSelectors
}

func parseContainerLogFilenames(directoryPath string, containerLogFilenames []string) []ContainerLog {

	var containerLogs []ContainerLog

	//container log files are in format PODNAME_NAMESPACE_CONTAINERNAME-CONTAINERUID.log
	//TODO check that CONTAINERNAME and CONTAINERUID are correct terminology
	for _, logFile := range containerLogFilenames {
		logfileSplitOnDot := strings.Split(logFile, ".")
		logFileSplitOnUnderscore := strings.Split(logfileSplitOnDot[0], "_")
		containerNameWithIDSplitOnDash := strings.Split(logFileSplitOnUnderscore[2], "-")

		//uid is the last value
		indexOfUid := len(containerNameWithIDSplitOnDash) - 1

		//containerName is everything except the last value, joined
		containerName := strings.Join(containerNameWithIDSplitOnDash[0:indexOfUid], "")

		containerLogs = append(containerLogs, ContainerLog{
			podname:       logFileSplitOnUnderscore[0],
			namespace:     logFileSplitOnUnderscore[1],
			containerName: containerName,
			containeruid:  containerNameWithIDSplitOnDash[indexOfUid],
			filepath:      path.Join(directoryPath, logFile),
		})
	}

	return containerLogs
}

func getAllContainerLogFilesThatHaveEverRunOnHost() ([]string, error) {
	output, err := utils.RunCommandOnHost("ls", containerLogDirectory)
	if err != nil {
		return nil, err
	}

	containers := strings.Split(output, "\n")
	containers = containers[:len(containers)-1]

	return containers, nil
}
