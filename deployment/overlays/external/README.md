# External Overlay (Deprecated)

This overlay produces the Periscope resource specification for the production images in MCR. The output of this can be consumed by external tools, like VS Code and AZ CLI.

**NOTE**: The preferred approach for consuming tools is to use `Kustomize` directly. See [main notes](../../../README.md#dependent-consuming-tools-and-working-contract).

The storage account data is not known at this time. The consuming tools are responsible for substituting all configuration data into the output, so this ensures we produce well-known placeholders for the various settings.

```sh
# Important: set the desired MCR version tag
export IMAGE_TAG=...
export SAS_KEY_PLACEHOLDER="# <saskey, base64 encoded>"
export ACCOUNT_NAME_PLACEHOLDER="# <accountName, string>"
export CONTAINER_NAME_PLACEHOLDER="# <containerName, string>"
# In the kustomize output, the placeholder will be base-64 encoded.
# Work out what it will be, so we can replace it with our desired placeholder.
sas_key_env_var_b64=$(echo -n '${SAS_KEY_PLACEHOLDER}' | base64)
kubectl kustomize ./deployment/overlays/external | sed -e "s/$sas_key_env_var_b64/$SAS_KEY_PLACEHOLDER/g" | envsubst
```