#!/bin/bash

echo
echo 1. DNS Setup
kubectl -n aks-periscope get apd aks-periscope-diagnostic -o jsonpath="{.spec.dns}" | jq  -r '["Level", "NameServer", "Custom"] as $fields | $fields, (.[] | [(.[$fields[]]|@json)]) | @tsv' | column -t
echo
echo
echo 2. Network Outbound Check
kubectl -n aks-periscope get apd aks-periscope-diagnostic -o jsonpath="{.spec.networkoutbound}" | jq  -r '["Type", "Start", "End", "Error"] as $fields | $fields, (.[] | [(.[$fields[]]|@json)]) | @tsv' | column -t
echo