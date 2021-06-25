package collector

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// ContainerLogsCollectorContainerD defines a ContainerLogs Collector struct for containerd clusters
type ContainerLogsCollectorContainerD struct {
	BaseCollector
}

var _ interfaces.Collector = &ContainerLogsCollectorContainerD{}

// NewContainerLogsCollectorContainerD is a constructor
func NewContainerLogsCollectorContainerD(exporters []interfaces.Exporter) *ContainerLogsCollectorContainerD {
	return &ContainerLogsCollectorContainerD{
		BaseCollector: BaseCollector{
			collectorType: ContainerLogsContainerD,
			exporters:     exporters,
		},
	}
}

type ContainerLog struct {
	podname string
	namespace string
	containerName string
	containeruid string
	filepath string
}

type ContainerLogSelector struct {
	namespace string
	containerNamePrefix string
}

// NOTE pod log files not currently used are in sub-directories of /var/log/pods and have format NAMESPACE_PODNAME_SOMEGUIDMEANINGTBD/CONTAINERNAME/#.log,
// where I assume # is incremented and a new log created for each container restart (starts at # == 0)
const containerLogDirectory = "/var/log/containers"

// Collect implements the interface method
func (collector *ContainerLogsCollectorContainerD) Collect() error {
	selectorStrings := strings.Fields(os.Getenv("DIAGNOSTIC_CONTAINERLOGS_LIST"))
	rootPath, err := utils.CreateCollectorDir(collector.GetName())

	containerLogSelectors := collector.ParseContainerLogSelectors(selectorStrings)

	allContainersEverRun, err := collector.GetAllContainerLogFilesThatHaveEverRunOnHost()
	if err != nil {
		return err
	}

	containerLogs := collector.ParseContainerLogFilenames(containerLogDirectory, allContainersEverRun)

	containerLogsToCollect := collector.DetermineContainerLogsToCollect(containerLogs, containerLogSelectors)

	for _, containerLog := range containerLogsToCollect {
		output, err := utils.RunCommandOnHost("cat", containerLog.filepath)
		containerLogOnContainer := filepath.Join(rootPath, filepath.Base(containerLog.filepath))

		err = utils.WriteToFile(containerLogOnContainer, output)
		if err != nil {
			return err
		}
		collector.AddToCollectorFiles(containerLogOnContainer)
	}

	return nil
}

//DetermineContainerLogsToCollect applies the containerLogSelectors to filter the list of containerLogs to be collected
func (collector *ContainerLogsCollectorContainerD) DetermineContainerLogsToCollect(allContainers []ContainerLog, selectors []ContainerLogSelector) []ContainerLog {
	var selectedContainerLogs []ContainerLog
	for _, containerLog := range allContainers {
		for _, selector := range selectors {
			if collector.DoesSelectorSelectContainerLog(containerLog, selector){
				selectedContainerLogs = append(selectedContainerLogs, containerLog)
			}
		}
	}
	return selectedContainerLogs
}

//DoesSelectorSelectContainerLog contains the logic for determining if a selector selects a containerLog for collecting
func (collector *ContainerLogsCollectorContainerD) DoesSelectorSelectContainerLog(containerLog ContainerLog, selector ContainerLogSelector) bool {
	return containerLog.namespace == selector.namespace && strings.HasPrefix(containerLog.containerName, selector.containerNamePrefix)
}

//ParseContainerLogSelectors parses selectorStrings into component struct
//TODO allow the raw struct objects to be defined directly in the deployment yaml and add additional required logic to DoesSelectorSelectContainerLog
func (collector *ContainerLogsCollectorContainerD) ParseContainerLogSelectors(selectorStrings []string) []ContainerLogSelector {
	var containerLogSelectors []ContainerLogSelector

	for _, selectorString := range selectorStrings {
		selectorStringParts := strings.Split(selectorString, "/")

		if len(selectorStringParts) == 1{
		containerLogSelectors = append(containerLogSelectors, ContainerLogSelector{
			namespace:           selectorStringParts[0],
		})}
		if len(selectorStringParts) == 2{
		containerLogSelectors = append(containerLogSelectors, ContainerLogSelector{
			namespace:           selectorStringParts[0],
			containerNamePrefix: selectorStringParts[1],
		})}
	}

	return containerLogSelectors
}

//ParseContainerLogFilenames parses container log filenames into component struct
func (collector *ContainerLogsCollectorContainerD) ParseContainerLogFilenames(directoryPath string, containerLogFilenames []string) []ContainerLog {

	var containerLogs []ContainerLog

	//container log files are in format PODNAME_NAMESPACE_CONTAINERNAME-CONTAINERUID.log
	//TODO check that CONTAINERNAME and CONTAINERUID are correct terminology
	for _, logFile := range containerLogFilenames {
		logfileSplitOnDot := strings.Split(logFile, ".")
		logFileSplitOnUnderscore := strings.Split(logfileSplitOnDot[0], "_")
		containerNameWithIDSplitOnDash := strings.Split(logFileSplitOnUnderscore[2], "-")

		//uid is the last value
		indexOfUid := len(containerNameWithIDSplitOnDash)-1

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

//GetAllContainerLogFilesThatHaveEverRunOnHost gets the list of log files for all containers that have ever run on the host
func (collector *ContainerLogsCollectorContainerD) GetAllContainerLogFilesThatHaveEverRunOnHost() ([]string, error){
	output, err := utils.RunCommandOnHost("ls", containerLogDirectory)
	if err != nil {
		return nil, err
	}

	containers := strings.Split(output, "\n")
	containers = containers[:len(containers)-1]

	return containers, nil
}
