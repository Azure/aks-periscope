package action

import (
	"path/filepath"
	"time"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

type kubeletCmdAction struct {
	name                     string
	collectIntervalInSeconds int
	processIntervalInSeconds int
	exportIntervalInSeconds  int
	exporter                 interfaces.Exporter
}

var _ interfaces.Action = &kubeletCmdAction{}

// NewKubeletCmdAction is a constructor
func NewKubeletCmdAction(collectIntervalInSeconds int, processIntervalInSeconds int, exportIntervalInSeconds int, exporter interfaces.Exporter) interfaces.Action {
	return &kubeletCmdAction{
		name:                     "kubeletcmd",
		collectIntervalInSeconds: collectIntervalInSeconds,
		processIntervalInSeconds: processIntervalInSeconds,
		exportIntervalInSeconds:  exportIntervalInSeconds,
		exporter:                 exporter,
	}
}

// GetName implements the interface method
func (action *kubeletCmdAction) GetName() string {
	return action.name
}

// Collect implements the interface method
func (action *kubeletCmdAction) Collect() ([]string, error) {
	rootPath, _ := utils.CreateCollectorDir(action.GetName())
	kubeletcmdFile := filepath.Join(rootPath, action.GetName())

	go func(kubeletcmdFile string) {
		ticker := time.NewTicker(time.Duration(action.collectIntervalInSeconds) * time.Second)
		for ; true; <-ticker.C {
			collectKubeletCmd(kubeletcmdFile)
		}
	}(kubeletcmdFile)

	return []string{kubeletcmdFile}, nil
}

// Process implements the interface method
func (action *kubeletCmdAction) Process(collectFiles []string) ([]string, error) {
	return nil, nil
}

// Export implements the interface method
func (action *kubeletCmdAction) Export(exporter interfaces.Exporter, collectFiles []string, processfiles []string) error {
	if exporter != nil {
		return exporter.Export(append(collectFiles, processfiles...), action.exportIntervalInSeconds)
	}

	return nil
}

func collectKubeletCmd(kubeletcmdFile string) error {
	output, _ := utils.RunCommandOnHost("ps", "-o", "cmd=", "-C", "kubelet")
	err := utils.WriteToFile(kubeletcmdFile, output)
	if err != nil {
		return err
	}

	return nil
}
