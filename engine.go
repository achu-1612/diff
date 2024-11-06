package diff

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DiffEnging is the entrypoint for the diff package.
type DiffEngine struct {
	handlers       map[string]FileHandler // File extension to handler mapping
	defaultHandler FileHandler
	config         *Configuration
	logger         *Logger
	mu             sync.RWMutex
}

// NewDiffEngine creates a new DiffEngine instance.
func NewDiffEngine(config *Configuration) (*DiffEngine, error) {
	if config == nil {
		config = DefaultConfig()
	}

	logger, err := NewLogger(config.DetailedLogging, "diff.log")
	if err != nil {
		return nil, err
	}

	engine := &DiffEngine{
		handlers: make(map[string]FileHandler),
		config:   config,
		logger:   logger,
	}

	engine.initializeHandlers()
	return engine, nil
}

// initializeHandlers initializes the default handlers.
// Note: For now we only have a generic binary handler and a text file handler.
// TODO: Add more handlers for different file types.
func (e *DiffEngine) initializeHandlers() {
	e.defaultHandler = NewGenericBinaryHandler()

	e.RegisterHandler(".txt", &TextFileHandler{})
	e.RegisterHandler(".log", &TextFileHandler{})
	e.RegisterHandler(".md", &TextFileHandler{})
}

// RegisterHandler registers a new file handler for a specific file extension.
// This can be used to add custom handlers for different file types.
func (e *DiffEngine) RegisterHandler(ext string, handler FileHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.handlers[ext] = handler
}

// getHandler returns the file handler for a specific file extension.
func (e *DiffEngine) getHandler(filename string) FileHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ext := strings.ToLower(filepath.Ext(filename))
	if handler, ok := e.handlers[ext]; ok {
		return handler
	}
	return e.defaultHandler
}

// CompareDirs compares two directories and returns differences
func (e *DiffEngine) CompareDirs(oldDir, newDir string) (*DiffSummary, []DiffResult, error) {
	summary := &DiffSummary{
		FileTypes: make(map[string]int),
		StartTime: time.Now(),
	}

	var results []DiffResult
	var mutex sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, e.config.Concurrency)

	// Process new and modified files
	err := filepath.Walk(newDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check file size limit
		if info.Size() > e.config.MaxFileSizeBytes {
			e.logger.Log("Skipping large file: %s (size: %d bytes)", path, info.Size())
			return nil
		}

		relPath, err := filepath.Rel(newDir, path)
		if err != nil {
			return err
		}

		// Check ignore patterns
		for _, pattern := range e.config.IgnorePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				return nil
			}
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(path, relPath string, info os.FileInfo) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			oldPath := filepath.Join(oldDir, relPath)
			result, err := e.compareFiles(oldPath, path, info)
			if err != nil {
				e.logger.Log("Error comparing files %s: %v", relPath, err)
				return
			}

			if result != nil {
				mutex.Lock()
				results = append(results, *result)
				summary.TotalFiles++

				switch result.Operation {
				case "added":
					summary.AddedFiles++
				case "modified":
					summary.ModifiedFiles++
				}

				summary.TotalSizeBytes += info.Size()

				if result.IsCompressed {
					summary.CompressedBytes += int64(len(result.Chunks[0].NewData))
				}

				summary.FileTypes[result.FileType]++
				mutex.Unlock()
			}
		}(path, relPath, info)

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	wg.Wait()

	// Check for deleted files
	err = filepath.Walk(oldDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(oldDir, path)
		if err != nil {
			return err
		}

		newPath := filepath.Join(newDir, relPath)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			summary.DeletedFiles++
			summary.TotalFiles++
			results = append(results, DiffResult{
				Path:      relPath,
				Operation: "deleted",
				OldHash:   calculateHash(path),
				ModTime:   info.ModTime(),
				Size:      info.Size(),
			})
		}

		return nil
	})

	summary.EndTime = time.Now()
	return summary, results, err
}

// compareFiles compares two files and returns the difference
func (e *DiffEngine) compareFiles(oldPath, newPath string, newInfo os.FileInfo) (*DiffResult, error) {
	oldData, err := os.ReadFile(oldPath)
	if os.IsNotExist(err) {
		newData, err := os.ReadFile(newPath)
		if err != nil {
			return nil, err
		}

		return &DiffResult{
			Path:         filepath.Base(newPath),
			Operation:    "added",
			NewHash:      calculateHash(newPath),
			FileType:     e.getHandler(newPath).GetFileType(),
			Size:         newInfo.Size(),
			ModTime:      newInfo.ModTime(),
			Permissions:  newInfo.Mode(),
			IsCompressed: e.config.CompressPatches,
			Chunks: []DiffChunk{{
				Offset:    0,
				NewData:   compressData(newData, e.config.CompressPatches, e.config.CompressionLevel),
				ChunkType: e.getHandler(newPath).GetFileType(),
			}},
		}, nil
	} else if err != nil {
		return nil, err
	}

	newData, err := os.ReadFile(newPath)
	if err != nil {
		return nil, err
	}

	if bytes.Equal(oldData, newData) {
		return nil, nil
	}

	handler := e.getHandler(newPath)
	chunks, err := handler.Compare(oldData, newData)
	if err != nil {
		return nil, err
	}

	if len(chunks) == 0 {
		return nil, nil
	}

	// Compress chunks if enabled
	if e.config.CompressPatches {
		for i := range chunks {
			chunks[i].NewData = compressData(chunks[i].NewData, true, e.config.CompressionLevel)
		}
	}

	return &DiffResult{
		Path:         filepath.Base(newPath),
		Operation:    "modified",
		OldHash:      calculateHash(oldPath),
		NewHash:      calculateHash(newPath),
		Chunks:       chunks,
		FileType:     handler.GetFileType(),
		Size:         newInfo.Size(),
		ModTime:      newInfo.ModTime(),
		Permissions:  newInfo.Mode(),
		IsCompressed: e.config.CompressPatches,
	}, nil
}
