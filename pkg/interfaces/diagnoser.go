package interfaces

// Diagnoser defines interface for a diagnoser
type Diagnoser interface {
	GetName() string

	Diagnose() error

	Export() error
}
