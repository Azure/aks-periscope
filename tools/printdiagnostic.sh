#!/bin/bash

echo
echo 1. Network Setup
for NODEAPD in $(kubectl -n aks-periscope get apd -o name)
do
    kubectl -n aks-periscope get $NODEAPD -o jsonpath="{.spec.networkconfig}" | jq .
done

echo
echo 2. Network Outbound Check
for NODEAPD in $(kubectl -n aks-periscope get apd -o name)
do
    kubectl -n aks-periscope get $NODEAPD -o jsonpath="{.spec.networkoutbound}" | jq .
    echo
done