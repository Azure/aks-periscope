# Dynamic Image Overlay Template

This is a template for an overlay, rather than an overlay itself, because although `Kustomize` supports dynamic configuration for `ConfigMap` and `Secret` resources via `.env` files, it does not allow dynamically specifying image names/tags.

This allows us to specify image/tag identifiers as well as runtime configuration, generating an overlay in the `overlays/temp` folder. This overlay can then be deployed to any cluster (including `Kind` and AKS).

## Image Sources

Some uses for this template are listed below.

### CI Build

The [CI Pipeline](../../../.github/workflows/ci-pipeline.yaml) builds an image accessible only to a local `Kind` cluster. The generated overlay deploys Periscope resources that reference this image.

### GHCR

It can be useful to test a particular GitHub branch. We can generate an overlay for deploying the image generated from that branch.

To make both the Docker image available to the cluster, it must be published to a container registry that allows anonymous pull access. To do this:
1. Push the branch you want to deploy to your local fork of the Periscope repository.
2. Run the [Building and Pushing to GHCR](../../../.github/workflows/build-and-publish.yml) workflow in GitHub Actions (making sure to select the correct branch).
3. Take note of the published image tag (e.g. '0.0.8').
4. [First time only] Under Package Settings in GitHub, set the package visibility to 'public'.

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
export IMAGE_NAME=ghcr.io/${REPO_USERNAME}/aks/periscope
```

We then generate a clean overlay in `overlays/temp`:

```sh
rm -rf ./deployment/overlays/temp && mkdir ./deployment/overlays/temp
touch ./deployment/overlays/temp/.env.config

cat ./deployment/overlays/dynamic-image/kustomization.template.yaml | envsubst > ./deployment/overlays/temp/kustomization.yaml
```

And finally deploy the resources:

```sh
# Ensure kubectl has the right cluster context
export KUBECONFIG=...
# Deploy
kubectl apply -k ./deployment/overlays/temp
```
