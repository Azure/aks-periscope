# Inspektor Gadget Integration

## develop locally

It follows closely to the original developer approach. On top of kind cluster, a managed cluster can be used as well. 
Be sure to allow the cluster to access the container image repository. In case of AKS and ACR, run 
```bash
az aks update --name <cluster-name> --resource-group <resource-group> --attach-acr <repo>
```

This section summarises the commands in order:
```shell
# after making your development changes
go build ./...
docker build -f ./builder/Dockerfile.linux -t periscope-local .
docker tag periscope-local <repo>.azurecr.io/periscope-local:<tag>
docker push <repo>.azurecr.io/periscope-local:<tag>
```
update deployment/overlays/dev/kustomization.yaml to use the image such that 
```yaml
images:
- name: periscope-linux
  newName: <repo>.azurecr.io/periscope-local
  newTag: <tag>
```

deploy the new version into your cluster
```shell
k apply -k ./deployment/overlays/dev
runId=$(date -u '+%Y-%m-%dT%H-%M-%SZ')
k patch configmap -n aks-periscope-dev diagnostic-config -p="{\"data\":{\"DIAGNOSTIC_RUN_ID\": \"$runId\"}}"
```

you can watch the logs of the periscope pods:
```shell
k -n aks-periscope-dev -l app=aks-periscope logs -f
```
The logs should show that files are collected and written to the designated storage account. 


## Chaos 

The [folder](pkg/test/resources/tools-resources/chaos) contains configuration that can break your cluster. 
Use this to see whether the inspektor gadget can point out the problem.