# Deploy with Kustomize

To store the logs an Azure Blob Service account is required.

Patch the DeamonSet to add the `AZURE_BLOB_ACCOUNT_NAME` env var:

```yaml
patches:
- target:
    group: apps
    kind: DaemonSet
    name: aks-periscope
    version: v1
  patch: |-
    - op: add
      path: '/spec/template/spec/containers/0/env/-'
      value:
        name: AZURE_BLOB_ACCOUNT_NAME
        value: your_account_name
```

## Connect to the Storage Account using a SAS key

Create the following secret to connect to the Storage Account using a SAS Key:

```yaml
secretGenerator:
- name: azureblob-secret
  literals:
  - AZURE_BLOB_SAS_KEY=your_sas_key_base_64_encoded

patches:
- target:
    group: apps
    kind: DaemonSet
    name: aks-periscope
    version: v1
  patch: |-
    - op: add
      path: '/spec/template/spec/containers/0/envFrom/-'
      value: |
        secretRef:
          name: azureblob-secret
```

## Apply

```sh
kubectl apply -f <(kustomize build)
```
