package collector

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
)

// OnDemandLogsCollector defines a OnDemandLogs Collector struct
type OnDemandLogsCollector struct {
	BaseCollector
}

var _ interfaces.Collector = &OnDemandLogsCollector{}

// NewOnDemandLogsCollector is a constructor
func NewOnDemandLogsCollector(exporter interfaces.Exporter) *OnDemandLogsCollector {
	return &OnDemandLogsCollector{
		BaseCollector: BaseCollector{
			collectorType: OnDemandLogs,
			exporter:      exporter,
		},
	}
}

// Collect implements the interface method
func (collector *OnDemandLogsCollector) Collect() error {
	onDemandLogs := strings.Fields(os.Getenv("DIAGNOSTIC_ONDEMANDLOGS_LIST"))
	rootPath, err := utils.CreateCollectorDir(collector.GetName())
	if err != nil {
		return err
	}

	for _, onDemandLog := range onDemandLogs {
		normalizedOnDemandLog := strings.Replace(onDemandLog, "/", "_", -1)
		if normalizedOnDemandLog[0] == '_' {
			normalizedOnDemandLog = normalizedOnDemandLog[1:]
		}

		onDemandLogFile := filepath.Join(rootPath, normalizedOnDemandLog)

		output, err := utils.RunCommandOnHost("cat", onDemandLog)
		if err != nil {
			return err
		}

		err = utils.WriteToFile(onDemandLogFile, output)
		if err != nil {
			return err
		}

		collector.AddToCollectorFiles(onDemandLogFile)
	}

	return nil
}
