#/bin/bash
set +x
set +e
set -o pipefail
set -u

###########################################################################
# Usage: bash -f ./stop.sh
#        Supported Options -
#               --kind-cluster-name=<Kind Cluster Name> (default INTEGRATION_TEST_CLUSTER)
###########################################################################

# All Supported Arguments
ARGUMENT_LIST=(
    "kind-cluster-name"
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
        *)
            break
            ;;
    esac
done

# Assign Deafults
CLUSTER_NAME=${CLUSTER_NAME:-"INTEGRATION_TEST_CLUSTER"}

echo $(date -u) "[INFO] Deleting a KIND cluster ${CLUSTER_NAME}"
./kind delete cluster --name ${CLUSTER_NAME}