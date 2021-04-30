package diagnoser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

type configValidatorDiagnosticDatum struct {
	HostName string `json:"HostName"`
	CRDName  string `json:CRDName`
}

// CustomValidatorDiagnoser defines a CustomValidator Diagnoser struct
type ConfigValidatorDiagnoser struct {
	BaseDiagnoser
	customResourceCollector *collector.CustomResourceCollector
}

var _ interfaces.Diagnoser = &ConfigValidatorDiagnoser{}

// NewNetworkConfigDiagnoser is a constructor
func NewConfigValidatorDiagnoser(customResourceCollector *collector.CustomResourceCollector, exporter interfaces.Exporter) *ConfigValidatorDiagnoser {
	return &ConfigValidatorDiagnoser{
		BaseDiagnoser: BaseDiagnoser{
			diagnoserType: ConfigValidator,
			exporter:      exporter,
		},
		customResourceCollector: customResourceCollector,
	}
}
func (diagnoser *ConfigValidatorDiagnoser) Diagnose() error {
	hostName, err := utils.GetHostName()
	rootPath, err := utils.CreateDiagnosticDir()
	if err != nil {
		return err
	}
	configValidatorDiagnosticFile := filepath.Join(rootPath, diagnoser.GetName())
	f, err := os.OpenFile(configValidatorDiagnosticFile, os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		return fmt.Errorf("Fail to open file %s: %+v", configValidatorDiagnosticFile, err)
	}

	configValidatorDiagnosticData := configValidatorDiagnosticDatum{HostName: hostName}

	diagnoser.AddToDiagnoserFiles(configValidatorDiagnosticFile)

	err = utils.WriteToCRD(configValidatorDiagnosticFile, diagnoser.GetName())
	if err != nil {
		return fmt.Errorf("Fail to write file %s to CRD: %+v", configValidatorDiagnosticFile, err)
	}
	return nil
}
