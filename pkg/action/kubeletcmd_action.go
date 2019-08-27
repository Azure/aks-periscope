package action

import (
	"os"
	"path/filepath"

	"github.com/Azure/aks-diagnostic-tool/pkg/interfaces"
	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// KubeletCmdAction defines an action on kubelet cmd
type KubeletCmdAction struct{}

var _ interfaces.Action = &KubeletCmdAction{}

// GetName implements the interface method
func (action *KubeletCmdAction) GetName() string {
	return "kubeletcmd"
}

// Collect implements the interface method
func (action *KubeletCmdAction) Collect() ([]string, error) {
	rootPath, _ := utils.CreateCollectorDir(action.GetName())
	output, _ := utils.RunCommandOnHost("ps", "-o", "cmd=", "-C", "kubelet")

	kubeletcmdFile := filepath.Join(rootPath, action.GetName())
	file, _ := os.Create(kubeletcmdFile)
	defer file.Close()

	file.Write([]byte(output))
	return []string{kubeletcmdFile}, nil
}

// Process implements the interface method
func (action *KubeletCmdAction) Process([]string) error {
	return nil
}
