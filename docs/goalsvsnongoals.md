# Goal vs Non-Goals

## Goals:

The goal of `AKS Periscope` is to allow `AKS` customers to run initial diagnostics and collect and export the logs (like into an Azure Blob storage account) to help them analyse and identify potential problems. AKS cluster issues are many times are caused by wrong configuration of their environment, such as networking or permission issues. This tool will allow `AKS` customers to run initial diagnostics and collect logs and custom analyses that help them identify the underlying problems.

The logs does this tool collects are documented here - https://github.com/Azure/aks-periscope#overview

## Non-Goals: 

This tool is written with `AKS` in mind and does not cover any broader k8s distros or services. In the broader OSS tool system, there are many OSS tools which provide better support for other scenarios, but this tool is `AKS` specific.
 