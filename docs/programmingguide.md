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