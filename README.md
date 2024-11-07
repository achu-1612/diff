## Diff

`diff` is a Go package that provides functionality to compare files and directories, identify differences, and generate patches. It supports both text and binary files and can handle large directories with configurable concurrency.

### Features

- Compare text and binary files
- Compare directories recursively
- Generate detailed diff summaries
- Configurable concurrency for large directory comparisons
- Customizable file handlers for different file types
- Logging support

### Installation

To install the package, run:

```sh
go get github.com/achu-1612/diff
```

### Usage

#### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "github.com/achu-1612/diff"
)

func main() {
    config := diff.DefaultConfig()
    engine, err := diff.NewDiffEngine(config)
    if err != nil {
        log.Fatalf("Failed to create diff engine: %v", err)
    }

    oldDir := "path/to/old/dir"
    newDir := "path/to/new/dir"

    summary, results, err := engine.CompareDirs(oldDir, newDir)
    if err != nil {
        log.Fatalf("Failed to compare directories: %v", err)
    }

    fmt.Printf("Summary: %+v\n", summary)
    for _, result := range results {
        fmt.Printf("Result: %+v\n", result)
    }
}
```

#### Custom File Handlers

You can register custom file handlers for specific file types:

```go
package main

import (
    "github.com/achu-1612/diff"
)

type CustomFileHandler struct{}

func (h *CustomFileHandler) Compare(old, new []byte) ([]diff.DiffChunk, error) {
    // Custom comparison logic
    return nil, nil
}

func (h *CustomFileHandler) Patch(original []byte, chunks []diff.DiffChunk) ([]byte, error) {
    // Custom patching logic
    return nil, nil
}

func (h *CustomFileHandler) GetFileType() string {
    return "custom"
}

func main() {
    config := diff.DefaultConfig()
    engine, err := diff.NewDiffEngine(config)
    if err != nil {
        log.Fatalf("Failed to create diff engine: %v", err)
    }

    engine.RegisterHandler(".custom", &CustomFileHandler{})
}
```

### Configuration

The `Configuration` struct allows you to customize the behavior of the diff engine:

```go
type Configuration struct {
    Concurrency         int
    MaxFileSizeBytes    int64
    IgnorePatterns      []string
    CompressPatches     bool
    CompressionLevel    int
    DetailedLogging     bool
}

func DefaultConfig() *Configuration {
    return &Configuration{
        Concurrency:      4,
        MaxFileSizeBytes: 10 * 1024 * 1024, // 10 MB
        IgnorePatterns:   []string{".git", "node_modules"},
        CompressPatches:  true,
        CompressionLevel: 5,
        DetailedLogging:  false,
    }
}
```

### Logging

The package includes a simple logger that can write to a file and/or stdout:

```go
logger, err := diff.NewLogger(true, "diff.log")
if err != nil {
    log.Fatalf("Failed to create logger: %v", err)
}
logger.Log("This is a log message")
logger.Close()
```

### License

This project is licensed under the MIT License.
