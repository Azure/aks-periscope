module github.com/Azure/aks-periscope

// 1.16 required for go:embed (used for testing resources)
go 1.16

require (
	github.com/Azure/azure-storage-blob-go v0.14.0
	github.com/docker/docker v20.10.17+incompatible
	github.com/google/uuid v1.2.0
	github.com/hashicorp/go-multierror v1.1.1
	helm.sh/helm/v3 v3.10.3
	k8s.io/api v0.25.2
	k8s.io/apimachinery v0.25.2
	k8s.io/cli-runtime v0.25.2
	k8s.io/client-go v0.25.2
	k8s.io/kubectl v0.25.2
	k8s.io/metrics v0.25.2
)
