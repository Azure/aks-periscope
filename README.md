
# AKS Periscope
Quick troubleshooting for your Azure Kubernetes Service (AKS) cluster.

# Overview
Hopefully most of the time, your AKS cluster is running happily and healthy. However, when things do go wrong, AKS customers need a tool to help them diagnose and collect the logs necessary to troubleshoot the issue. It can be difficult to collect the appropriate node and pod logs to figure what's wrong, how to fix the problem, or even to pass on those logs to others to help. 

AKS Periscope allows AKS customers to run initial diagnostics and collect and export the logs (like into an Azure Blob storage account) to help them analyze and identify potential problems or easily share the information to support to help with the troubleshooting process with a simple az aks kollect command. These cluster issues are many times are caused by wrong configuration of their environment, such as networking or permission issues. This tool will allow AKS customers to run initial diagnostics and collect logs and custom analyses that helps them identify the underlying problems.

![Architecture](https://user-images.githubusercontent.com/33297523/64900285-f5b65c00-d644-11e9-9a52-c4345d1b1861.png)


# Data Privacy and Collection
AKS Periscope runs on customer's agent pool nodes, and collect VM and container level data. It is important that the customer is aware and gives consent before the tool is deployed/information shared. Microsoft guidelines can be found in the link below:

https://azure.microsoft.com/en-us/support/legal/support-diagnostic-information-collection/


# Compatibility
AKS Periscope currently only work on Linux based agent nodes. Please see https://github.com/PatrickLang/logslurp for Windows based agent nodes. 


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

It also generates the following diagnostic analyses:
1. Network outbound connectivity,  reports the down period for a specific connection.
2. DNS, check if customized DNS is used.


# How to Use
AKS Periscope can be deployed by using Azure Command-Line tool (CLI). The steps are:

0. If CLI extension aks-preview has been installed previously, uninstall it first.
```
az extension remove --name aks-preview
``` 

1. Install CLI extension aks-preview.
```
az extension add --name aks-preview
``` 

2. Run `az aks kollect` command to collect metrics and diagnostic information, and upload to an Azure storage account.

    1. If a storage account is already setup for the AKS cluster in Diagnostic Settings (https://docs.microsoft.com/en-us/azure/azure-monitor/platform/diagnostic-logs-stream-log-store), simply run the command below, and it automatically uses the existing storage account.
    ```
    az aks kollect -g myresourcegroup -n mycluster
    ```

    2. Specify a storage account and SAS token.
    ```
    az aks kollect -g myresourcegroup -n mycluster --storage-account mystorageaccount --sas-token mysastoken
    ```

    3. Specify a storage account resource ID.
    ```
    az aks kollect -g myresourcegroup -n mycluster --storage-account /subscriptions/xxx/resourceGroups/xxx/providers/Microsoft.Storage/storageAccounts/xxx
    ```


All collected logs, metrics and node level diagnostic information is stored on host nodes under directory:
```
/var/log/aks-periscope
```
This directory is also mounted to container as:
```
/aks-periscope
```
After export, they will also be stored in Azure Blob Service under a container with its name equals to cluster API server FQDN.

Alternatively, AKS Periscope can be deployed directly with `kubectl`. See instructions in Appendix.

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
