package collector

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/aks-periscope/pkg/interfaces"
	"github.com/Azure/aks-periscope/pkg/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	restclient "k8s.io/client-go/rest"
)

type HelmRelease struct {
	Name      string               `json:"name"`
	Namespace string               `json:"namespace"`
	Status    release.Status       `json:"status"`
	ChartName string               `json:"chart"`
	History   []HelmReleaseHistory `json:"history"`
}

type HelmReleaseHistory struct {
	Date       time.Time      `json:"lastDeployment"`
	Message    string         `json:"description"`
	Status     release.Status `json:"status"`
	Revision   int            `json:"revision"`
	AppVersion string         `json:"appVersion"`
}

// HelmCollector defines a Helm Collector struct
type HelmCollector struct {
	data        map[string]string
	kubeconfig  *restclient.Config
	runtimeInfo *utils.RuntimeInfo
}

// NewHelmCollector is a constructor
func NewHelmCollector(config *restclient.Config, runtimeInfo *utils.RuntimeInfo) *HelmCollector {
	return &HelmCollector{
		data:        make(map[string]string),
		kubeconfig:  config,
		runtimeInfo: runtimeInfo,
	}
}

func (collector *HelmCollector) GetName() string {
	return "helm"
}

func (collector *HelmCollector) CheckSupported() error {
	if !utils.Contains(collector.runtimeInfo.CollectorList, "connectedCluster") {
		return fmt.Errorf("not included because 'connectedCluster' not in COLLECTOR_LIST variable. Included values: %s", strings.Join(collector.runtimeInfo.CollectorList, " "))
	}

	return nil
}

// Implement the RESTClientGetter interface for helm client initialization.
// This allows us to provide our restclient.Config directly, without having
// to copy individual fields.
func (collector *HelmCollector) ToRESTConfig() (*restclient.Config, error) {
	return collector.kubeconfig, nil
}
func (collector *HelmCollector) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return nil, nil
}
func (collector *HelmCollector) ToRESTMapper() (meta.RESTMapper, error) {
	return nil, nil
}
func (collector *HelmCollector) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return nil
}

// Collect implements the interface method
func (collector *HelmCollector) Collect() error {
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(collector, "", "", log.Printf); err != nil {
		return fmt.Errorf("init action configuration: %w", err)
	}

	releases, err := action.NewList(actionConfig).Run()
	if err != nil {
		return fmt.Errorf("list helm releases: %w", err)
	}

	result := make([]HelmRelease, 0)

	for _, release := range releases {
		release.Chart.AppVersion()
		r := HelmRelease{
			Name:      release.Name,
			Namespace: release.Namespace,
			Status:    release.Info.Status,
			ChartName: release.Chart.Name(),
		}

		histories, err := action.NewHistory(actionConfig).Run(release.Name)

		if err != nil {
			log.Printf("Get release %s history failed: %v", release.Name, err)
		} else {
			r.History = make([]HelmReleaseHistory, 0)
			for _, history := range histories {
				h := HelmReleaseHistory{
					Date:       history.Info.LastDeployed.Time,
					Message:    history.Info.Description,
					Status:     history.Info.Status,
					Revision:   history.Version,
					AppVersion: history.Chart.AppVersion(),
				}
				r.History = append(r.History, h)
			}
		}

		result = append(result, r)
	}

	b, err := json.Marshal(result)

	if err != nil {
		return fmt.Errorf("marshall helm releases to json: %w", err)
	}

	collector.data["helm_list"] = string(b)

	return nil
}

func (collector *HelmCollector) GetData() map[string]interfaces.DataValue {
	return utils.ToDataValueMap(collector.data)
}
