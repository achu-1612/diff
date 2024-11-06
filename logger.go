package diff

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Logger is a simple logger that can write to a file and/or stdout.
type Logger struct {
	detailed bool
	logFile  *os.File
	mu       sync.Mutex
}

// NewLogger creates a new Logger instance.
func NewLogger(detailed bool, logPath string) (*Logger, error) {
	var logFile *os.File
	var err error

	if logPath != "" {
		logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
	}

	return &Logger{
		detailed: detailed,
		logFile:  logFile,
	}, nil
}

// Log writes a log message to the logger.
func (l *Logger) Log(format string, args ...interface{}) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))

	if l.logFile != nil {
		l.logFile.WriteString(msg)
	}

	if l.detailed {
		fmt.Print(msg)
	}
}

func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}
