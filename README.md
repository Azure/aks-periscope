
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
Link to [Current Feature Set].


# User Guide
Link to [User Guide].


# Programming Guide
Link to [Programming Guide].


# Feedback 
Feel free to contact aksperiscope@microsoft.com or open an issue with any feedback or questions about AKS Periscope. This is currently a work in progress, but look out for more capabilities to come!


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

[Current Feature Set]: docs/features.md
[User Guide]: docs/userguide.md
[Programming Guide]: docs/programmingguide.md