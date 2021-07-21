package collector

import (
	. "github.com/onsi/gomega"
	"strings"
	"testing"
)

var parseContainerLogSelectorTests = []struct {
	selectorString       []string
	containerLogSelector []ContainerLogSelector
}{
	{[]string{"kube-system"}, []ContainerLogSelector{{namespace: "kube-system", containerNamePrefix: ""}}},
	{[]string{"--all-namespaces"}, []ContainerLogSelector{{namespace: "--all-namespaces", containerNamePrefix: ""}}},
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
	g := NewWithT(t)
	for _, tt := range parseContainerLogSelectorTests {
		testName := strings.Join(tt.selectorString, " ")
		t.Run(testName, func(t *testing.T) {

			selectors := parseContainerLogSelectors(tt.selectorString)

			for i, selector := range selectors {
				g.Expect(selector).To(Equal(tt.containerLogSelector[i]))
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
	g := NewWithT(t)
	for _, tt := range ParseContainerLogFilenamesTests {
		testName := strings.Join(tt.containerLogFilenames, " ")
		t.Run(testName, func(t *testing.T) {
			containerLogs := parseContainerLogFilenames(containerLogDirectory, tt.containerLogFilenames)

			for i, containerLog := range containerLogs {
				g.Expect(containerLog.filepath).To(Equal(tt.containerLogs[i].filepath))
			}
		})
	}
}

var (
	DoesSelectorSelectContainerLogTests = []struct {
		testName             string
		containerLog         ContainerLog
		containerLogSelector ContainerLogSelector
		selected             bool
	}{
		{
			testName: "just-namespace-selects-true-correctly",
			containerLog: ContainerLog{
				podname:       "kube-apiserver-kind-control-plane",
				namespace:     "kube-system",
				containerName: "kube-apiserver",
				containeruid:  "341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f",
				filepath:      "/var/log/containers/kube-apiserver-kind-control-plane_kube-system_kube-apiserver-341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f.log",
			},
			containerLogSelector: ContainerLogSelector{
				namespace:           "kube-system",
				containerNamePrefix: "",
			},
			selected: true,
		},
		{
			testName: "just-namespace-selects-false-correctly",
			containerLog: ContainerLog{
				podname:       "kube-apiserver-kind-control-plane",
				namespace:     "kube-system",
				containerName: "kube-apiserver",
				containeruid:  "341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f",
				filepath:      "/var/log/containers/kube-apiserver-kind-control-plane_kube-system_kube-apiserver-341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f.log",
			},
			containerLogSelector: ContainerLogSelector{
				namespace:           "default",
				containerNamePrefix: "",
			},
			selected: false,
		},
		{
			testName: "namespace-and-containernameprefix-selects-true-correctly",
			containerLog: ContainerLog{
				podname:       "kube-apiserver-kind-control-plane",
				namespace:     "kube-system",
				containerName: "kube-apiserver",
				containeruid:  "341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f",
				filepath:      "/var/log/containers/kube-apiserver-kind-control-plane_kube-system_kube-apiserver-341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f.log",
			},
			containerLogSelector: ContainerLogSelector{
				namespace:           "kube-system",
				containerNamePrefix: "kube-api",
			},
			selected: true,
		},
		{
			testName: "namespace-match-and-containernameprefix-nonMatch-selects-false-correctly",
			containerLog: ContainerLog{
				podname:       "kube-apiserver-kind-control-plane",
				namespace:     "kube-system",
				containerName: "kube-apiserver",
				containeruid:  "341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f",
				filepath:      "/var/log/containers/kube-apiserver-kind-control-plane_kube-system_kube-apiserver-341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f.log",
			},
			containerLogSelector: ContainerLogSelector{
				namespace:           "kube-system",
				containerNamePrefix: "kube-sched",
			},
			selected: false,
		},
		{
			testName: "namespace-nonMatch-and-containernameprefix-match-selects-false-correctly",
			containerLog: ContainerLog{
				podname:       "kube-apiserver-kind-control-plane",
				namespace:     "kube-system",
				containerName: "kube-apiserver",
				containeruid:  "341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f",
				filepath:      "/var/log/containers/kube-apiserver-kind-control-plane_kube-system_kube-apiserver-341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f.log",
			},
			containerLogSelector: ContainerLogSelector{
				namespace:           "default",
				containerNamePrefix: "kube-api",
			},
			selected: false,
		},
		{
			testName: "--all-namespaces-selects-true-correctly",
			containerLog: ContainerLog{
				podname:       "kube-apiserver-kind-control-plane",
				namespace:     "kube-system",
				containerName: "kube-apiserver",
				containeruid:  "341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f",
				filepath:      "/var/log/containers/kube-apiserver-kind-control-plane_kube-system_kube-apiserver-341694b9f16b51d7afc7c0a68d2ea44f31f6c2dad550d56d8d8dd9304a27b01f.log",
			},
			containerLogSelector: ContainerLogSelector{
				namespace:           "--all-namespaces",
				containerNamePrefix: "",
			},
			selected: true,
		},
	}
)

// TestDoesSelectorSelectContainerLog test container log selector parser
func TestDoesSelectorSelectContainerLog(t *testing.T) {
	g := NewWithT(t)
	for _, tt := range DoesSelectorSelectContainerLogTests {
		testName := tt.testName
		t.Run(testName, func(t *testing.T) {
			selected := doesSelectorSelectContainerLog(tt.containerLog, tt.containerLogSelector)

			g.Expect(selected).To(Equal( tt.selected))
		})
	}
}

//TODO tests for DetermineContainerLogsToCollect
