package interfaces

// Action defines interface for an action
type Action interface {
	GetName() string

	Collect() ([]string, error)

	Process([]string) ([]string, error)

	Export(Exporter, []string, []string) error
}
