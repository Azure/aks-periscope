# User Guide

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