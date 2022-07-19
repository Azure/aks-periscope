package interfaces

// DataProducer defines an object producing data
type DataProducer interface {
	GetData() map[string]DataValue

	GetName() string
}
