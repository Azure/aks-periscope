package collector

import (
	"strings"
	"testing"
)

var parseContainerLogSelectorTests = []struct {
	selectorString       []string
	containerLogSelector []ContainerLogSelector
}{
	{[]string{"kube-system"}, []ContainerLogSelector{{namespace: "kube-system", containerNamePrefix: ""}}},
	{[]string{"kube-system/metrics-agent"}, []ContainerLogSelector{{namespace: "kube-system", containerNamePrefix: "metrics-agent"}}},
	{[]string{
		"kube-system/metrics-agent",
		"azure-arc/fluent-bit"},
		[]ContainerLogSelector{
			{namespace: "kube-system", containerNamePrefix: "metrics-agent"},
			{namespace: "azure-arc", containerNamePrefix: "fluent-bit"}}},
}

// TestParseContainerLogSelectors test container log selector parser
func TestParseContainerLogSelectors(t *testing.T) {
	for _, tt := range parseContainerLogSelectorTests {
		testName := strings.Join(tt.selectorString, " ")
		t.Run(testName, func(t *testing.T) {
			selectors := parseContainerLogSelectors(tt.selectorString)

			for i, selector := range selectors {
				if selector != tt.containerLogSelector[i] {
					t.Errorf("parseContainerLogSelectors(%q) => %q, want %q",
						testName, selector, tt.containerLogSelector[i])
				}
			}
		})
	}
}

var ParseContainerLogFilenamesTests = []struct {
	directoryPath         string
	containerLogFilenames []string
	containerLogs         []ContainerLog
}{
	{containerLogDirectory, []string{"kube-apiserver-kind-control-plane_kube-system_kube-apiserver-341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f.log"},
		[]ContainerLog{{
			podname:       "kube-apiserver-kind-control-plane",
			namespace:     "kube-system",
			containerName: "kube-apiserver",
			containeruid:  "341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f",
			filepath:      "/var/log/containers/kube-apiserver-kind-control-plane_kube-system_kube-apiserver-341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f.log"}}},
	{containerLogDirectory, []string{
		"metrics-agent-5b9b94754f-lv4gm_azure-arc_metrics-agent-b2aa15b3f9c7c395539281bd94ee0725706c7228439a8870494804e042a54d7d.log",
		"resource-sync-agent-f8c7c6b6b-zdqkc_azure-arc_fluent-bit-46d805fc1d05986cce94f845401949c241fa577e4312df45e5bfb39b27dc226c.log"},

		[]ContainerLog{{
			podname:       "metrics-agent-5b9b94754f-lv4gm",
			namespace:     "azure-arc",
			containerName: "metrics-agent",
			containeruid:  "b2aa15b3f9c7c395539281bd94ee0725706c7228439a8870494804e042a54d7d",
			filepath:      "/var/log/containers/metrics-agent-5b9b94754f-lv4gm_azure-arc_metrics-agent-b2aa15b3f9c7c395539281bd94ee0725706c7228439a8870494804e042a54d7d.log"}, {

			podname:       "resource-sync-agent-f8c7c6b6b-zdqkc",
			namespace:     "azure-arc",
			containerName: "fluent-bit",
			containeruid:  "46d805fc1d05986cce94f845401949c241fa577e4312df45e5bfb39b27dc226c",
			filepath:      "/var/log/containers/resource-sync-agent-f8c7c6b6b-zdqkc_azure-arc_fluent-bit-46d805fc1d05986cce94f845401949c241fa577e4312df45e5bfb39b27dc226c.log"}}},
}

// TestParseContainerLogSelectors test container log selector parser
func TestParseContainerLogFilenames(t *testing.T) {
	for _, tt := range ParseContainerLogFilenamesTests {
		testName := strings.Join(tt.containerLogFilenames, " ")
		t.Run(testName, func(t *testing.T) {
			containerLogs := parseContainerLogFilenames(containerLogDirectory, tt.containerLogFilenames)

			for i, containerLog := range containerLogs {
				if containerLog.filepath != tt.containerLogs[i].filepath {
					t.Errorf("parseContainerLogFilenames(%q)=> %q, want %q",
						testName, containerLog.filepath, tt.containerLogs[i].filepath)
				}
			}
		})
	}
}

//TODO tests for DetermineContainerLogsToCollect and DoesSelectorSelectContainerLog
