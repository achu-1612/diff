package diff

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"
)

const (
	testLogFileName = "test_log.txt"
)

func TestLogger_Log(t *testing.T) {
	creatTestFile(t)
	defer cleanTestDir(t)

	tests := []struct {
		name     string
		detailed bool
		logPath  string
		message  string
		args     []interface{}
	}{
		{
			name:     "Log to stdout only",
			detailed: true,
			logPath:  "",
			message:  "Test message %d",
			args:     []interface{}{1},
		},
		{
			name:     "Log to file only",
			detailed: false,
			logPath:  testDatadir + "/" + testLogFileName,
			message:  "Test message %d",
			args:     []interface{}{2},
		},
		{
			name:     "Log to both stdout and file",
			detailed: true,
			logPath:  testDatadir + "/" + testLogFileName,
			message:  "Test message %d",
			args:     []interface{}{3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new logger
			logger, err := NewLogger(tt.detailed, tt.logPath)
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}

			// Capture stdout
			var stdout bytes.Buffer
			if tt.detailed {
				old := os.Stdout
				r, w, _ := os.Pipe()
				os.Stdout = w

				logger.Log(tt.message, tt.args...)

				w.Close()
				os.Stdout = old

				stdout.ReadFrom(r)
			} else {
				logger.Log(tt.message, tt.args...)
			}

			logger.Close()

			// Check log file if specified
			if tt.logPath != "" {
				defer os.Remove(tt.logPath)

				fileContent, err := os.ReadFile(tt.logPath)
				if err != nil {
					t.Fatalf("Failed to read log file: %v", err)
				}

				expected := "[" + time.Now().Format(time.RFC3339) + "] " + fmt.Sprintf(tt.message, tt.args...) + "\n"
				if !bytes.Contains(fileContent, []byte(expected)) {
					t.Errorf("Log file content = %s, want %s", fileContent, expected)
				}
			}

			// Check stdout if detailed logging is enabled
			if tt.detailed {
				expected := "[" + time.Now().Format(time.RFC3339) + "] " + fmt.Sprintf(tt.message, tt.args...) + "\n"
				if !bytes.Contains(stdout.Bytes(), []byte(expected)) {
					t.Errorf("Stdout content = %s, want %s", stdout.String(), expected)
				}
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	creatTestFile(t)
	defer cleanTestDir(t)

	tests := []struct {
		name      string
		detailed  bool
		logPath   string
		wantError bool
	}{
		{
			name:      "Logger with detailed output and valid log path",
			detailed:  true,
			logPath:   testDatadir + "/" + testLogFileName,
			wantError: false,
		},
		{
			name:      "Logger without detailed output and valid log path",
			detailed:  false,
			logPath:   testDatadir + "/" + testLogFileName,
			wantError: false,
		},
		{
			name:      "Logger with detailed output and empty log path",
			detailed:  true,
			logPath:   "",
			wantError: false,
		},
		{
			name:      "Logger without detailed output and empty log path",
			detailed:  false,
			logPath:   "",
			wantError: false,
		},
		{
			name:      "Logger with invalid log path",
			detailed:  true,
			logPath:   testDatadir + "/invalid/path/to/file",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.detailed, tt.logPath)
			if (err != nil) != tt.wantError {
				t.Fatalf("NewLogger() error = %v, wantError %v", err, tt.wantError)
			}

			if logger != nil && logger.logFile != nil {
				logger.logFile.Close()
				os.Remove(tt.logPath)
			}
		})
	}
}
