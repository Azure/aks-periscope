package interfaces

// Action defines interface for an action
type Action interface {
	GetName() string

	GetCollectIntervalInSeconds() int

	GetCollectCountForProcess() int

	GetCollectCountForExport() int

	Collect() error

	Process() error

	Export() error
}
