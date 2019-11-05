# Appendix

Alternatively, AKS Periscope can be deployed directly with `kubectl`. The steps are:

1. Download the deployment file:
```
deployment/aks-periscope.yaml
```

By default, the collected logs, metrics and node level diagnostic information will be exported to Azure Blob Service. An Azure Blob Service account and a Shared Access Signature (SAS) token need to be provisioned in advance. These values should be based64 encoded and be set in the `azureblob-secret` in above aks-periscope.yaml.

Additionally, to collect container logs and describe Kubenetes objects (pods and services) in namespaces beyond the default `kube-system`, user can configure the `containerlogs-config` and `kubeobjects-config` in above aks-periscope.yaml.

2. Deploy the daemon set using kubectl:
```
kubectl apply -f aks-periscope.yaml
```