
# AKS Diagnostic Tool
Quick troubleshooting your AKS cluster.

# Overview
AKS (Azure kubernetes Service) customers needs a tool to diagnose their cluster issues which many times are caused by wrong configuration of their environment, such as networking or permission issues. This tool will allow AKS customers to run initial diagnostics and collect logs that helps them identify the underlying problems.

# Build Instructions

The container image is currently stored in aksrepos.azurecr.io. A JIT ticket is needed and then login by:
```
az acr login -n aksrepos
```

Then build and push to ACR by:
```
docker build -f ./builder/Dockerfile -t staging/aks-diagnostic .
docker tag staging/aks-diagnostic aksrepos.azurecr.io/staging/aks-diagnostic:v0.1
docker push aksrepos.azurecr.io/staging/aks-diagnostic:v0.1
```

# Getting Started

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