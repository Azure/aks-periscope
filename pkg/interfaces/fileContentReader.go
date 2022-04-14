package interfaces

// FileContentReader defines interface for a collector
type FileContentReader interface {
	GetFileContent(filePath string) (string, error)
}
