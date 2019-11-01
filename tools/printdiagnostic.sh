#!/bin/bash

echo
echo 1. Network Setup
echo
for NODEAPD in $(kubectl -n aks-periscope get apd -o name)
do
    kubectl -n aks-periscope get $NODEAPD -o jsonpath="{.spec.networkconfig}" | jq  -r '["HostName", "NetworkPlugin", "VirtualMachineDNS", "KubernetesDNS", "MaxPodsPerNode"] as $fields | $fields, (. | [(.[$fields[]]|@json)]) | @tsv' | column -t
    echo
done
echo
echo
echo 2. Network Outbound Check
echo
for NODEAPD in $(kubectl -n aks-periscope get apd -o name)
do
    kubectl -n aks-periscope get $NODEAPD -o jsonpath="{.spec.networkoutbound}" | jq  -r '["HostName", "Type", "Start", "End", "Status"] as $fields | $fields, (.[] | [(.[$fields[]]|@json)]) | @tsv' | column -t
    echo
done
echo