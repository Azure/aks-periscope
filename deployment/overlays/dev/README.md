# Dev Overlay

This can be used for running a locally-built Linux Periscope image in a `Kind` cluster. Because `Kind` runs on Linux only, the Linux `DaemonSet` will refer to the locally-built image, whereas the Windows `DaemonSet` will refer to the latest published production Windows MCR image.

It will deploy to its own namespace, `aks-periscope-dev` to avoid conflicts with any existing Periscope deployment.

## Building a Local Image

First, build the image and make sure it's loaded in `Kind`. If it's not, the pod will fail trying to pull the image (because it's local).

```sh
docker build -f ./builder/Dockerfile.linux -t periscope-local .

# Include a --name argument here if not using the default kind cluster.
kind load docker-image periscope-local
```

## Setting up Configuration Data

To run correctly, Periscope requires some storage account configuration that is different for each user. It also has some optional 'diagnostic' configuration (node log locations, etc.).

We need to make sure this doesn't get into source control, so it is stored in `gitignore`d `.env` files.

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

# Set up configuration data for Kustomize
# (for further customization, the variables in the .env.config file can be configured to override the defaults)
touch ./deployment/overlays/dev/.env.config
cat <<EOF > ./deployment/overlays/dev/.env.secret
AZURE_BLOB_ACCOUNT_NAME=${stg_account}
AZURE_BLOB_SAS_KEY=?${sas}
AZURE_BLOB_CONTAINER_NAME=${blob_container}
EOF
```

## Deploying to Kind

Once the `.env` files are in place, `Kustomize` has all the information it needs to generate the `yaml` resource specification for Periscope.

```sh
# Ensure kubectl has the right cluster context
export KUBECONFIG=...

# Deploy
kubectl apply -k ./deployment/overlays/dev
```
