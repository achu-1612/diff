package diff

// FileHandler is an interface that defines the methods which can be used to compare and patch files.
type FileHandler interface {
	Compare(old, new []byte) ([]DiffChunk, error)
	Patch(original []byte, chunks []DiffChunk) ([]byte, error)
	GetFileType() string
}
