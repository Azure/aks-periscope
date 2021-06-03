package interfaces

// DataProducer defines an object producing data
type DataProducer interface {
	GetData() map[string]string

	GetName() string
}
