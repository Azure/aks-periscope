package collector

import (
	"io/ioutil"
	"os"
	"strings"
)

// NodeLogsCollector defines a NodeLogs Collector struct
type NodeLogsCollector struct {
	data map[string]string
}

// NewNodeLogsCollector is a constructor
func NewNodeLogsCollector() *NodeLogsCollector {
	return &NodeLogsCollector{
		data: make(map[string]string),
	}
}

func (collector *NodeLogsCollector) GetName() string {
	return "nodelogs"
}

// Collect implements the interface method
func (collector *NodeLogsCollector) Collect() error {
	nodeLogs := strings.Fields(os.Getenv("DIAGNOSTIC_NODELOGS_LIST"))

	for _, nodeLog := range nodeLogs {

		output, err := os.Open(nodeLog)
		if err != nil {
			return err
		}

		defer output.Close()

		b, err := ioutil.ReadAll(output)
		if err != nil {
			return err
		}

		collector.data["nodeLog"] = string(b)
	}

	return nil
}

func (collector *NodeLogsCollector) GetData() map[string]string {
	return collector.data
}
