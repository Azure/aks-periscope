package collector

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"k8s.io/apimachinery/pkg/util/wait"
)

const windowsLogsCollectorPrefix = "collect-windows-logs/"

type WindowsLogsCollector struct {
	data         map[string]interfaces.DataValue
	osIdentifier utils.OSIdentifier
	runtimeInfo  *utils.RuntimeInfo
	filePaths    *utils.KnownFilePaths
	fileSystem   interfaces.FileSystemAccessor
	pollInterval time.Duration
	timeout      time.Duration
}

func NewWindowsLogsCollector(osIdentifier utils.OSIdentifier, runtimeInfo *utils.RuntimeInfo, filePaths *utils.KnownFilePaths, fileSystem interfaces.FileSystemAccessor, pollInterval, timeout time.Duration) *WindowsLogsCollector {
	return &WindowsLogsCollector{
		data:         make(map[string]interfaces.DataValue),
		osIdentifier: osIdentifier,
		runtimeInfo:  runtimeInfo,
		filePaths:    filePaths,
		fileSystem:   fileSystem,
		pollInterval: pollInterval,
		timeout:      timeout,
	}
}

func (collector *WindowsLogsCollector) GetName() string {
	return "windowslogs"
}

func (collector *WindowsLogsCollector) CheckSupported() error {
	// This is specifically for Windows.
	if collector.osIdentifier != utils.Windows {
		return fmt.Errorf("unsupported OS: %s", collector.osIdentifier)
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
	err := wait.PollUntilContextTimeout(context.Background(), collector.pollInterval, collector.timeout, false,
		func(context.Context) (bool, error) {
			return collector.fileSystem.FileExists(completionNotificationPath)
		})

	if err != nil {
		return fmt.Errorf("error waiting for windows log collection: %w", err)
	}

	// We should now expect to find a 'logs' directory containing all the logs for this run.
	logsDirectory := path.Join(collector.filePaths.WindowsLogsOutput, "logs")
	logFilePaths, err := collector.fileSystem.ListFiles(logsDirectory)
	if err != nil {
		return fmt.Errorf("error listing files in %s: %w", logsDirectory, err)
	}

	for _, logFilePath := range logFilePaths {
		size, err := collector.fileSystem.GetFileSize(logFilePath)
		if err != nil {
			return fmt.Errorf("error getting file size %s: %w", logFilePath, err)
		}

		relativePath := windowsLogsCollectorPrefix + strings.TrimPrefix(logFilePath, logsDirectory+"/")
		collector.data[relativePath] = utils.NewFilePathDataValue(collector.fileSystem, logFilePath, size)
	}

	return nil
}

func (collector *WindowsLogsCollector) GetData() map[string]interfaces.DataValue {
	return collector.data
}
