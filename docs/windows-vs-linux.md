# Windows vs. Linux

AKS Periscope runs on both Windows and Linux nodes, but the information it collects differs between each OS. This is a summary of those differences.

## Collectors not enabled on Windows

The following collectors are currently unavailable on Windows:

- DNS: This relies on `resolv.conf`, which is unavailable in Windows.
- IPTables: The `iptables` command is not available on Windows.
- Kubelet: This shows the arguments used to invoke the kubelet process. Windows containers do not support shared process namespaces, and so we cannot see processes on the host node.
- SystemLogs: This uses `journalctl` to retrieve system logs, which is not available on Windows.

## Node Logs differences

Since Windows and Linux nodes have a completely different file structure, the files collected by the `NodeLogsCollector` differ between OS. These are configurable, but by default Periscope will collect:

**Linux**
- /var/log/azure/cluster-provision.log
- /var/log/cloud-init.log

**Windows**
- C:\AzureData\CustomDataSetupScript.log
