
# AKS Diagnostic Tool
Quick troubleshooting your AKS cluster.

# Overview
AKS (Azure kubernetes Service) customers needs a tool to diagnose their cluster issues which many times are caused by wrong configuration of their environment, such as networking or permission issues. This tool will allow AKS customers to run initial diagnostics and collect logs that helps them identify the underlying problems.

# How to Use
AKS Diagnostic Tool can be deployed as a daemon set on Kubernetes agent nodes. The steps are:

1. Download the deployment file: <br /> `deployment/aks-diagnostic.yaml` <br /> By default, the collected logs, metrics and node level diagnostic information will be exported to Azure Blob Service. An Azure Blob Service account and a Shared Access Signature (SAS) token need to be provisioned in advance. These values should be based64 encoded and be set in the in `azure-blob` secret in above aks-diagnostic.yaml.

2. Deploy the daemon set using kubectl: <br /> `kubectl apply -f aks-diagnostic.yaml`

3. All collected logs, metrics and node level diagnostic information is stored on host nodes under directory: <br /> `/var/log/aks-diagnostic` <br /> If exported, they will also be stored in Azure Blob Service under a container with its name equals to cluster API server FQDN.


# Programming Guide
AKS Diagnostic Tool provides a simple framework which supports adding new functionalities. The steps are:

1. Clone this repo.

2. Add a new *Action* golang file which implements Action interface defined in `pkg/interfaces/action.go`. *Action* denotes a plugable program unit of collecting and processing one kind of metrics. There are already a few *Action* implementations that can be found under: `pkg/action`, and a sample implementation can be found under: <br /> `pkg/action/containerlogs_action.go`

3. If additional way of exporting data is needed, An Exporter interface is also provided in `pkg/interfaces/exporter.go`, and a sample implementation can be found under: <br /> `pkg/exporter/azureblob_exporter.go`

4. Chain the newly implemented action in the main program: `cmd/aks-diagnostic/aks-diagnostic.go`.

5. For internal development, the container image is currently stored in aksrepos.azurecr.io. To connect to the ACR repo, a JIT ticket on microsoft tenant is needed, and then login by: <br /> `az acr login -n aksrepos` <br /> Then build container image, tag it, and push it to ACR: <br /> 
`docker build -f ./builder/Dockerfile -t staging/aks-diagnostic .` <br />
`docker tag staging/aks-diagnostic aksrepos.azurecr.io/staging/aks-diagnostic:test` <br />
`docker push aksrepos.azurecr.io/staging/aks-diagnostic:test` <br />

6. For external development, build container image and push it to a container registry.

7. Modify the deployment file `deployment/aks-diagnostic.yaml` to use the newly built image.


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