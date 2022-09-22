package collector

import (
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"

	restclient "k8s.io/client-go/rest"
)

// InspektorGadgetBlockIOCollector defines a InspektorGadget Top BlockIO Collector struct
type InspektorGadgetBlockIOCollector struct {
	data          map[string]string
	kubeconfig    *restclient.Config
	commandRunner *utils.KubeCommandRunner
	runtimeInfo   *utils.RuntimeInfo
}

// NewInspektorGadgetBlockIOCollector is a constructor.
func NewInspektorGadgetBlockIOCollector(config *restclient.Config, runtimeInfo *utils.RuntimeInfo) *InspektorGadgetBlockIOCollector {
	return &InspektorGadgetBlockIOCollector{
		data:          make(map[string]string),
		kubeconfig:    config,
		commandRunner: utils.NewKubeCommandRunner(config),
		runtimeInfo:   runtimeInfo,
	}
}

func (collector *InspektorGadgetBlockIOCollector) GetName() string {
	return "inspektorgadget-block_io"
}

func (collector *InspektorGadgetBlockIOCollector) CheckSupported() error {
	// TODO check whether gadget plugin is installed, for now assume so
	return nil
}

// Collect implements the interface method
func (collector *InspektorGadgetBlockIOCollector) Collect() error {
	output, err := utils.RunCommandOnHost("kubectl-gadget", "top", "block-io")
	if err != nil {
		return err
	}

	collector.data["block-io"] = output
	return nil
}

func (collector *InspektorGadgetBlockIOCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}
