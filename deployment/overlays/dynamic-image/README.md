# Dynamic Image Overlay Template

This is a template for an overlay, rather than an overlay itself, because although `Kustomize` supports dynamic configuration for `ConfigMap` and `Secret` resources via `.env` files, it does not allow dynamically specifying image names/tags.

This allows us to specify image/tag identifiers as well as runtime configuration, generating an overlay in the `overlays/temp` folder. This overlay can then be deployed to any cluster (including `Kind` and AKS).

## Image Sources

Some uses for this template are listed below.

### CI Build

The [CI Pipeline](../../../.github/workflows/ci-pipeline.yaml) builds an image accessible only to a local `Kind` cluster. The generated overlay deploys Periscope resources that reference this image.

### GHCR

It can be useful to test a particular GitHub branch. We can generate an overlay for deploying the images generated from that branch. This is especially useful for testing behaviour in Windows containers, because the typical development environment does not have `Docker` configured for running Windows containers (and if it does, the OS may not match the AKS Windows OS). Further notes on creating a cluster with Windows nodes is [below](#creating-a-windows-cluster).

To make both the Docker images available to the cluster, they must be published to a container registry that allows anonymous pull access. To do this:
1. Push the branch you want to deploy to your local fork of the Periscope repository.
2. Run the [Building and Pushing to GHCR](../../../.github/workflows/build-and-publish.yml) workflow in GitHub Actions (making sure to select the correct branch).
3. Take note of the published image tags (e.g. '0.0.8').
4. [First time only] Under Package Settings in GitHub, set each package's visibility to 'public'.

## Setting up Configuration Data

Like the `dev` overlay, we need to put storage account configuration into an `.env.secret` file before running `Kustomize`.

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

# Create a clean overlay folder
rm -rf ./deployment/overlays/temp && mkdir ./deployment/overlays/temp

# Set up storage configuration data for Kustomize
cat <<EOF > ./deployment/overlays/temp/.env.secret
AZURE_BLOB_ACCOUNT_NAME=${stg_account}
AZURE_BLOB_SAS_KEY=?${sas}
AZURE_BLOB_CONTAINER_NAME=${blob_container}
EOF
```

We can also override diagnostic configuration variables:

```sh
echo "DIAGNOSTIC_KUBEOBJECTS_LIST=kube-system default" > ./deployment/overlays/temp/.env.config
```

## Deploying Periscope

We first need to specify environment variables for image name and tag. For example, for GHCR:

```sh
REPO_USERNAME=...
export IMAGE_TAG=...
export IMAGE_NAME_LINUX=ghcr.io/${REPO_USERNAME}/aks/periscope
export IMAGE_NAME_WINDOWS=ghcr.io/${REPO_USERNAME}/aks/periscope-win
```

We then generate the `kustomization.yaml` and dependencies in `overlays/temp`:

```sh
touch ./deployment/overlays/temp/.env.config # In case it doesn't exist already
cat ./deployment/overlays/dynamic-image/kustomization.template.yaml | envsubst > ./deployment/overlays/temp/kustomization.yaml
```

And finally deploy the resources:

```sh
# Ensure kubectl has the right cluster context
export KUBECONFIG=...
# Deploy
kubectl apply -k ./deployment/overlays/temp
```

---

## Footnotes

### Creating a Windows Cluster

This section contains notes on creating a Windows cluster in AKS. It's documented here because creating a cluster with Windows nodes currently takes a little bit of command-line work.

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