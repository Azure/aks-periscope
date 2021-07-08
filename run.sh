#!/bin/bash
 
set -e
 
DOCKER_ACCOUNT="johshmsft"
DOCKER_REPO="aks-periscope"
 
echo "[****************] DOCKER BUILD AND PUSH [****************]"
docker build -f ./builder/Dockerfile -t "$DOCKER_ACCOUNT/$DOCKER_REPO" .
docker push "$DOCKER_ACCOUNT/$DOCKER_REPO"
echo ""

AKS_PERISCOPE_NS="aks-periscope"
AKS_PERISCOPE_DEPL_YAML_PATH="deployment/aks-periscope.yaml"
 
echo "[****************] KUBECTL DELETE [****************]"
kubectl delete --force -f "$AKS_PERISCOPE_DEPL_YAML_PATH" || true
echo ""

sleep 7

echo "[****************] KUBECTL APPLY [****************]"
kubectl apply --force -f "$AKS_PERISCOPE_DEPL_YAML_PATH"
echo ""
 
pods=$(kubectl get pods -n aks-periscope | grep --color=none aks-periscope | awk '{print $1}')
echo "[****************] AKS PERISCOPE PODS [****************]"
echo "${pods[@]}"
echo ""
 
for p in ${pods[@]}; do
    echo "[****************] LOGS FOR POD $p [****************]"
    kubectl wait --for=condition=ready pod "$p" -n "$AKS_PERISCOPE_NS"
    sleep 5
    kubectl logs "$p" -n "$AKS_PERISCOPE_NS"
    echo ""
done
 
