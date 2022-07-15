package interfaces

// FileContentReader defines interface for a collector
type FileContentReader interface {
	GetFileContent(filePath string) (string, error)
	FileExists(filePath string) (bool, error)
	ListFiles(directoryPath string) ([]string, error)
}
