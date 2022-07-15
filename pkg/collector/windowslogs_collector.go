package collector

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"k8s.io/apimachinery/pkg/util/wait"
)

type WindowsLogsCollector struct {
	data         map[string]string
	runtimeInfo  *utils.RuntimeInfo
	filePaths    *utils.KnownFilePaths
	fileReader   interfaces.FileContentReader
	pollInterval time.Duration
	timeout      time.Duration
}

func NewWindowsLogsCollector(runtimeInfo *utils.RuntimeInfo, filePaths *utils.KnownFilePaths, fileReader interfaces.FileContentReader, pollInterval, timeout time.Duration) *WindowsLogsCollector {
	return &WindowsLogsCollector{
		data:         make(map[string]string),
		runtimeInfo:  runtimeInfo,
		filePaths:    filePaths,
		fileReader:   fileReader,
		pollInterval: pollInterval,
		timeout:      timeout,
	}
}

func (collector *WindowsLogsCollector) GetName() string {
	return "windowslogs"
}

func (collector *WindowsLogsCollector) CheckSupported() error {
	// This is specifically for Windows.
	if collector.runtimeInfo.OSIdentifier != "windows" {
		return fmt.Errorf("unsupported OS: %s", collector.runtimeInfo.OSIdentifier)
	}

	// Even for Windows, this is only supported on kubernetes v1.23 or higher. It is up to consumers
	// to deploy the resources needed to support this. To ensure consumers have explicitly specified
	// this to run, we check for a well-known runtime variable.
	if !collector.runtimeInfo.HasFeature(utils.WindowsHpc) {
		return fmt.Errorf("feature not set: %s", utils.WindowsHpc)
	}

	// This relies on us having a known 'run ID'.
	if len(collector.runtimeInfo.RunId) == 0 {
		return errors.New("diagnostic run ID not set")
	}

	return nil
}

// Collect implements the interface method
func (collector *WindowsLogsCollector) Collect() error {
	// Exporting the logs is done by a separate process, which will place an empty file in a known
	// location to indicate completion. The name of that file is the current 'run ID'.
	completionNotificationPath := path.Join(collector.filePaths.WindowsLogsOutput, collector.runtimeInfo.RunId)

	// Poll to check existence of this file.
	err := wait.Poll(collector.pollInterval, collector.timeout, func() (bool, error) {
		return collector.fileReader.FileExists(completionNotificationPath)
	})

	if err != nil {
		return fmt.Errorf("error waiting for windows log collection: %w", err)
	}

	// We should now expect to find a 'logs' directory containing all the logs for this run.
	logsDirectory := path.Join(collector.filePaths.WindowsLogsOutput, "logs")
	logFilePaths, err := collector.fileReader.ListFiles(logsDirectory)
	if err != nil {
		return fmt.Errorf("error listing files in %s: %w", logsDirectory, err)
	}

	for _, logFilePath := range logFilePaths {
		content, err := collector.fileReader.GetFileContent(logFilePath)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", logFilePath, err)
		}

		relativePath := strings.TrimPrefix(logFilePath, logsDirectory+"/")
		collector.data[relativePath] = content
	}

	return nil
}

func (collector *WindowsLogsCollector) GetData() map[string]string {
	return collector.data
}
