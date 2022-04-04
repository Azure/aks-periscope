# GHCR Overlay

This can be used to deploy any version of Periscope on a GitHub branch to any cluster (including `Kind` and AKS). It is especially useful for testing behaviour in Windows containers, because the typical development environment does not have `Docker` configured for running Windows containers (and if they do, their OS may not match the AKS Windows OS).

It will deploy to its own namespace, `aks-periscope-ghcr` to avoid conflicts with any existing Periscope deployment.

## Creating a Windows AKS Cluster

This section can be skipped if not testing Windows containers, or a Windows cluster already exists. It's documented here because creating a cluster with Windows nodes currently takes a little bit of command-line work.

```sh
# Variables for subscription ID, resource group, cluster name and node-pool name
# node pool "may only contain lowercase alphanumeric characters and must begin with a lowercase letter"
sub_id=...
rg=...
aks_name=...
nodepool_name=...

# Create the cluster with a system nodepool (Linux)
az aks create \
    --subscription $sub_id \
    --resource-group $rg \
    --name $aks_name \
    --node-count 2 \
    --enable-addons monitoring \
    --generate-ssh-keys \
    --windows-admin-username WindowsUser1 \
    --vm-set-type VirtualMachineScaleSets \
    --network-plugin azure

# Create an additional user nodepool (Windows)
az aks nodepool add \
    --subscription $sub_id \
    --resource-group $rg \
    --cluster-name $aks_name \
    --os-type Windows \
    --name $nodepool_name \
    --node-count 1

# Set the kubectl context to the new cluster
az aks get-credentials \
    --subscription $sub_id \
    --resource-group $rg \
    --name $aks_name
```

## Publishing Images to GHCR

To make both the Windows and Linux images available to the cluster, they must be published to a container registry that allows anonymous pull access. The easiest way to do this is:
1. Push the branch you want to deploy to your local fork of the Periscope repository.
2. Run the [Building and Pushing to GHCR](../../../.github/workflows/build-and-publish.yml) workflow in GitHub Actions (making sure to select the correct branch).
3. Take note of the published image tag (e.g. '0.0.8').
4. [First time only] Under Package Settings in GitHub, set the package visibility to 'public'.

## Setting up Configuration Data

Like the `dev` overlay, we need to put storage account configuration into an `.env` file before running `Kustomize`.

```sh
# Create a SAS
sub_id=...
stg_account=...
blob_container=...
sas_expiry=`date -u -d "30 minutes" '+%Y-%m-%dT%H:%MZ'`
sas=$(az storage account generate-sas \
    --account-name $stg_account \
    --subscription $sub_id \
    --permissions rwdlacup \
    --services b \
    --resource-types sco \
    --expiry $sas_expiry \
    -o tsv)

# Set up storage configuration data for Kustomize
cat <<EOF > ./deployment/overlays/dev/.env.secret
AZURE_BLOB_ACCOUNT_NAME=${stg_account}
AZURE_BLOB_SAS_KEY=?${sas}
AZURE_BLOB_CONTAINER_NAME=${blob_container}
EOF
```

## Deploying Periscope with GHCR Images

The commands below generate the `yaml` for deploying to a cluster, substitute in the variables needed to use the images in GHCR, and apply it to the cluster.

```sh
# Environment variables for the repo username (fork) and image version (e.g. '0.0.8')
export REPO_USERNAME=...
export IMAGE_VERSION=...

# Ensure the kubectl context is set before running this:
kubectl kustomize ./deployment/overlays/ghcr | envsubst | kubectl apply -f -
```
