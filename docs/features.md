# Current Feature Set

1. Container logs (by default in the `kube-system` namespace)
2. Docker and Kubelet system service logs
3. Network outbound connectivity, include checks for internet, API server, Tunnel, ACR and MCR.
4. Node IP tables
5. Node Provision logs
6. Node and Kubernetes level DNS settings
7. Describe Kubernetes pods and services (by default in the `kube-system` namespace)
8. Kubelet command arguments.

It also generates the following diagnostic analyses:
1. Network outbound connectivity,  reports the down period for a specific connection.
2. DNS, check if customized DNS is used.