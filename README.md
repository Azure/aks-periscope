
# AKS Periscope
Quick troubleshooting your AKS cluster.

# Overview
AKS (Azure kubernetes Service) customers needs a tool to diagnose their cluster issues which many times are caused by wrong configuration of their environment, such as networking or permission issues. This tool will allow AKS customers to run initial diagnostics and collect logs that helps them identify the underlying problems.

![Architecture](https://user-images.githubusercontent.com/33297523/64049272-210b5800-cb29-11e9-9182-9b2a7b178c36.png)


# Data Privacy and Collection
AKS Periscope runs on customer's agent pool nodes, and collect VM and container level data. It is important that customer is aware and gives consent before the tool is deployed. Microsoft guidelines can be found in the link below:

https://azure.microsoft.com/en-us/support/legal/support-diagnostic-information-collection/


# Compatibility
AKS Periscope currently only work on Linux based agent nodes.


# Current Feature Set
Currently this tool collects the following metrics:
1. Container logs (by default in the `kube-system` namespace)
2. Docker and Kubelet system service logs
3. Network outbound connectivity, include checks for internet, API server, Tunnel, ACR and MCR.
4. Node IP tables
5. Node Provision logs
6. Node and Kubernetes level DNS settings
7. Describe Kubernetes pods and services (by default in the `kube-system` namespace)
8. Kubelet command arguments.

It also generates the following diagnostics:
1. Network outbound connectivity,  reports the down period for a specific connection.
2. DNS, check if customized DNS is used.


# How to Use
AKS Periscope is deployed as a daemon set on Kubernetes agent nodes. The steps are:

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

3. All collected logs, metrics and node level diagnostic information is stored on host nodes under directory:
```
/var/log/aks-periscope
```
This directory is also mounted to container as:
```
/aks-periscope
```
If exported, they will also be stored in Azure Blob Service under a container with its name equals to cluster API server FQDN.


# Programming Guide
AKS Periscope provides a simple framework which supports adding new functionalities. The steps are:

1. Clone this repo.

2. Add a new *Action* golang file which implements Action interface defined in `pkg/interfaces/action.go`. *Action* denotes a plugable program unit of collecting and processing one kind of metrics. There are already a few *Action* implementations that can be found under: `pkg/action`, and a sample implementation can be found under:
```
pkg/action/containerlogs_action.go
```

3. If additional way of exporting data is needed, An Exporter interface is also provided in `pkg/interfaces/exporter.go`, and a sample implementation can be found under:
```
pkg/exporter/azureblob_exporter.go
```

4. Chain the newly implemented action in the main program: `cmd/aks-periscope/aks-periscope.go`.

5. For internal development, the container image is currently stored in aksrepos.azurecr.io. To connect to the ACR repo, a JIT ticket on microsoft tenant is needed, and then login by:
```
az acr login -n aksrepos
```
Then build container image, tag it, and push it to ACR:
```
docker build -f ./builder/Dockerfile -t staging/aks-periscope .
docker tag staging/aks-periscope aksrepos.azurecr.io/staging/aks-periscope:test
docker push aksrepos.azurecr.io/staging/aks-periscope:test
```

6. For external development, build container image and push it to a container registry.

7. Modify the deployment file `deployment/aks-periscope.yaml` to use the newly built image.


# Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.