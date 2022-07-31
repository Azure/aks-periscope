package interfaces

// Collector defines interface for a collector
type Collector interface {
	GetName() string

	CheckSupported() error

	Collect() error

	GetData() map[string]DataValue
}
