#/bin/bash
set +x
set -e

###########################################################################
# Usage: bash -f ./start.sh
#        Supported Options -
#               --kind-cluster-name=<Kind Cluster Name> (default INTEGRATION_TEST_CLUSTER)
#               --kind-version (default v0.7.0)
#               --kubectl-version (default v1.18.0)
#               --helm-version (default v3.2.0)
###########################################################################

# All Supported Arguments
ARGUMENT_LIST=(
    "kind-cluster-name"
    "kind-version"
    "kubectl-version"
    "helm-version"
)

# Read Arguments
opts=$(getopt \
    --longoptions "$(printf "%s:," "${ARGUMENT_LIST[@]}")" \
    --name "$(basename "$0")" \
    --options "" \
    -- "$@"
)

# Assign Values from Arguments
eval set --$opts
while [[ $# -gt 0 ]]; do
    case "$1" in
        --kind-cluster-name)
            CLUSTER_NAME=$2
            shift 2
            ;;
        --kind-version)
            KIND_VERSION=$2
            shift 2
            ;;
        --kubectl-version)
            KUBECTL_VERSION=$2
            shift 2
            ;;
        --helm-version)
            HELM_VERSION=$2
            shift 2
            ;;
        *)
            break
            ;;
    esac
done

# Assign Deafults
CLUSTER_NAME=${CLUSTER_NAME:-"INTEGRATION_TEST_CLUSTER"}
KIND_VERSION=${KIND_VERSION:-v0.7.0}
KUBECTL_VERSION=${KUBECTL_VERSION:-v1.18.0}
HELM_VERSION=${HELM_VERSION:-v3.2.0}

echo $(date -u) "[INFO] Downloading Kubectl ..."
curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
chmod +x ./kubectl

echo $(date -u) "[INFO] Downloading KIND ..."
curl -Lo ./kind https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-$(uname)-amd64
chmod +x ./kind

echo $(date -u) "[INFO] Downloading helm ..."
wget https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz
tar -zxvf helm-${HELM_VERSION}-linux-amd64.tar.gz

echo $(date -u) "[INFO] Creating a KIND cluster ${CLUSTER_NAME} ..."
./kind create cluster --name ${CLUSTER_NAME}

echo $(date -u) "[INFO] Sleeping for 60s to make sure KIND cluster is ready to accept request ..."
sleep 60s