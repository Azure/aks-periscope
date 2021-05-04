package diagnoser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/collector"
	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

type configValidatorDiagnosticDatum struct {
	HostName    string `json:"HostName"`
	CRDName     string `json:CRDName`
	CRCondition string `json:CRCondition`
	CRMessage   string `json:CRMessage`
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

	configValidatorDiagnosticData := []configValidatorDiagnosticDatum{}
	for _, file := range diagnoser.customResourceCollector.GetCollectorFiles() {
		filename := strings.Split(file, "/")
		t, err := os.Open(file)
		defer t.Close()
		if err != nil {
			return fmt.Errorf("Fail to open file %s: %+v", file, err)
		}

		dataPoint := configValidatorDiagnosticDatum{HostName: hostName}
		scanner := bufio.NewScanner(t)
		isNameSet := false
		for scanner.Scan() {
			s := strings.Split(scanner.Text(), "\n")
			log.Printf(s[0])
			s[0] = strings.Trim(s[0], " ")

			if !isNameSet && strings.Contains(s[0], "Name:") {
				crd := strings.Split(s[0], " ")
				directories := strings.Split(filename[len(filename)-1], "_")
				dataPoint.CRDName = directories[0] + "_" + crd[len(crd)-1]
				isNameSet = true
			} else if strings.Contains(s[0], "Type:") {
				condition := strings.Split(s[0], " ")
				dataPoint.CRCondition = condition[len(condition)-1]
			} else if strings.Contains(s[0], "Kind:") {
				message := strings.Split(s[0], " ")
				dataPoint.CRMessage = message[len(message)-1]

			}

		}
		configValidatorDiagnosticData = append(configValidatorDiagnosticData, dataPoint)

	}

	dataBytes, err := json.Marshal(configValidatorDiagnosticData)
	if err != nil {
		return fmt.Errorf("Fail to marshal data: %+v", err)
	}

	_, err = f.WriteString(string(dataBytes))
	if err != nil {
		return fmt.Errorf("Fail to write data to file: %+v", err)
	}
	diagnoser.AddToDiagnoserFiles(configValidatorDiagnosticFile)

	err = utils.WriteToCRD(configValidatorDiagnosticFile, diagnoser.GetName())
	if err != nil {
		return fmt.Errorf("Fail to write file %s to CRD: %+v", configValidatorDiagnosticFile, err)
	}
	return nil
}
