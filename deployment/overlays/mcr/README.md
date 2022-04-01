# MCR Overlay

This overlay produces the Periscope resource specification for the production images in MCR. The output of this will be consumed by other tools, like VS Code and AZ CLI.

**NOTE**: If the consuming tools are altered so that they use `Kustomize` directly, this overlay will not be needed.

The storage account data is not known at this time. The consuming tools are responsible for substituting all configuration data into the output, so this ensures we produce well-known placeholders for the various settings.

```sh
# Important: set the desired MCR version tag
export IMAGE_VERSION=...

export SAS_KEY_PLACEHOLDER="# <saskey, base64 encoded>"
export ACCOUNT_NAME_PLACEHOLDER="# <accountName, string>"
export CONTAINER_NAME_PLACEHOLDER="# <containerName, string>"

# In the kustomize output, the placeholder will be base-64 encoded.
# Work out what it will be, so we can replace it with our desired placeholder.
sas_key_env_var_b64=$(echo -n '${SAS_KEY_PLACEHOLDER}' | base64)

kubectl kustomize ./deployment/overlays/mcr | sed -e "s/$sas_key_env_var_b64/$sas_key_placeholder/g" | envsubst
```
