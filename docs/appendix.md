# Appendix

Alternatively, AKS Periscope can be deployed directly with `kubectl`. The steps are:

1. Download the deployment file:
```
deployment/aks-periscope.yaml
```

By default, the collected logs, metrics and node level diagnostic information will be exported to Azure Blob Service. An Azure Blob Service account and a Shared Access Signature (SAS) token need to be provisioned in advance. These values should be based64 encoded and be set in the `AZURE_BLOB_ACCOUNT_NAME` and `AZURE_BLOB_SAS_KEY` in above aks-periscope.yaml.

   * `AZURE_BLOB_ACCOUNT_NAME` holds a base64 encoded storage account ID (e.g. "mystorageaccountname"). 
   * `AZURE_BLOB_SAS_KEY` holds a base64 encoded **Account level** SAS token granting the following permissions: ss=b srt=sco sp=rwdlacup (description of params here: https://docs.microsoft.com/en-us/rest/api/storageservices/create-account-sas#specifying-account-sas-parameters). 
       * this should be just the **query string** component of the SAS key, e.g. "?sv=2019-12-12&ss=btqf&...." not the full uri. 
       * Azure Storage Explorer or the Azure Portal can be used to generate the SAS.

Base64 encoding can be performed on linux via:
echo -n "string-to-encode" | base64

Additionally, to collect container logs and describe Kubenetes objects (pods and services) in namespaces beyond the default `kube-system`, user can configure the `containerlogs-config` and `kubeobjects-config` in above aks-periscope.yaml.

2. If Periscope has been previously deployed to the cluster, it will need to be manually removed first or the "kubectl apply" command below will succeed, but Periscope will silently fail to run:
```
kubectl delete -f aks-periscope.yaml
```

3. Deploy the daemon set using kubectl:
```
kubectl apply -f aks-periscope.yaml
```
