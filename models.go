package diff

import (
	"compress/gzip"
	"os"
	"time"
)

const Version = "1.0.0"

// Main types
type DiffResult struct {
	Path         string
	Operation    string // "added", "modified", "deleted"
	OldHash      string
	NewHash      string
	Chunks       []DiffChunk
	FileType     string
	Size         int64
	ModTime      time.Time
	Permissions  os.FileMode
	IsCompressed bool
}

type DiffChunk struct {
	Offset    int64
	OldData   []byte
	NewData   []byte
	ChunkType string // "binary", "text", "image"
}

type DiffSummary struct {
	TotalFiles      int
	AddedFiles      int
	ModifiedFiles   int
	DeletedFiles    int
	TotalSizeBytes  int64
	CompressedBytes int64
	FileTypes       map[string]int
	StartTime       time.Time
	EndTime         time.Time
}

// Configuration
type Configuration struct {
	CompressPatches     bool
	CompressionLevel    int
	ChunkSize           int64
	Concurrency         int
	IgnorePatterns      []string
	IncludePatterns     []string
	PreservePermissions bool
	MaxFileSizeBytes    int64
	BackupFiles         bool
	BackupDir           string
	DetailedLogging     bool
}

func DefaultConfig() *Configuration {
	return &Configuration{
		CompressPatches:     true,
		CompressionLevel:    gzip.BestCompression,
		ChunkSize:           1024 * 1024, // 1MB chunks
		Concurrency:         4,
		PreservePermissions: true,
		MaxFileSizeBytes:    1024 * 1024 * 100, // 100MB
		BackupFiles:         true,
		DetailedLogging:     false,
	}
}
