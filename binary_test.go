package diff

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"
)

func TestGenericBinaryHandler_Compare(t *testing.T) {
	tests := []struct {
		name     string
		old      []byte
		new      []byte
		wantDiff bool
		wantErr  bool
	}{
		{
			name:     "identical content",
			old:      []byte("hello world"),
			new:      []byte("hello world"),
			wantDiff: false,
			wantErr:  false,
		},
		{
			name:     "completely different content",
			old:      []byte("hello world"),
			new:      []byte("goodbye world"),
			wantDiff: true,
			wantErr:  false,
		},
		{
			name:     "empty files",
			old:      []byte{},
			new:      []byte{},
			wantDiff: false,
			wantErr:  false,
		},
		{
			name:     "one empty file",
			old:      []byte("hello world"),
			new:      []byte{},
			wantDiff: true,
			wantErr:  false,
		},
		{
			name:     "partial modification",
			old:      []byte("hello world 123"),
			new:      []byte("hello mars 123"),
			wantDiff: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewGenericBinaryHandler()
			chunks, err := h.Compare(tt.old, tt.new)

			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantDiff && len(chunks) == 0 {
				t.Error("Compare() expected diff chunks but got none")
			}

			if !tt.wantDiff && len(chunks) > 0 {
				t.Error("Compare() expected no diff chunks but got some")
			}

			// Verify patch reconstruction
			if len(chunks) > 0 {
				patched, err := h.Patch(tt.old, chunks)
				if err != nil {
					t.Errorf("Patch() error = %v", err)
					return
				}
				if !bytes.Equal(patched, tt.new) {
					t.Error("Patch() failed to reconstruct original content")
				}
			}
		})
	}
}

func TestGenericBinaryHandler_OptimizeBinaryDiff(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		wantMinMatch   int
		wantMaxGapSize int
	}{
		{
			name:           "high entropy data",
			data:           generateRandomBytes(1000),
			wantMinMatch:   16,
			wantMaxGapSize: 256,
		},
		{
			name:           "low entropy data",
			data:           bytes.Repeat([]byte("abcdef"), 1000),
			wantMinMatch:   4,
			wantMaxGapSize: 2048,
		},
		{
			name:           "large file",
			data:           bytes.Repeat([]byte("x"), 2*1024*1024),
			wantMinMatch:   8,
			wantMaxGapSize: 2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewGenericBinaryHandler()
			h.OptimizeBinaryDiff(tt.data)

			if h.MinMatchLength != tt.wantMinMatch {
				t.Errorf("OptimizeBinaryDiff() MinMatchLength = %v, want %v",
					h.MinMatchLength, tt.wantMinMatch)
			}

			if h.MaxGapSize != tt.wantMaxGapSize {
				t.Errorf("OptimizeBinaryDiff() MaxGapSize = %v, want %v",
					h.MaxGapSize, tt.wantMaxGapSize)
			}
		})
	}
}

func TestGenericBinaryHandler_AnalyzeBinaryDiff(t *testing.T) {
	tests := []struct {
		name           string
		old            []byte
		new            []byte
		wantMatchCount int
		wantEntropy    float64
	}{
		{
			name:           "identical content",
			old:            []byte("hello world"),
			new:            []byte("hello world"),
			wantMatchCount: 1,
			wantEntropy:    0.0, // approximate
		},
		{
			name:           "high entropy content",
			old:            generateRandomBytes(1000),
			new:            generateRandomBytes(1000),
			wantMatchCount: 0,
			wantEntropy:    0.8, // approximate
		},
		{
			name:           "repeating content",
			old:            bytes.Repeat([]byte("abc"), 100),
			new:            bytes.Repeat([]byte("abc"), 100),
			wantMatchCount: 1,
			wantEntropy:    0.3, // approximate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewGenericBinaryHandler()
			stats, err := h.AnalyzeBinaryDiff(tt.old, tt.new)
			if err != nil {
				t.Errorf("AnalyzeBinaryDiff() error = %v", err)
				return
			}

			if stats.MatchCount != tt.wantMatchCount {
				t.Errorf("AnalyzeBinaryDiff() MatchCount = %v, want %v",
					stats.MatchCount, tt.wantMatchCount)
			}

			// Allow for some margin of error in entropy calculation
			if stats.Entropy < tt.wantEntropy-0.2 || stats.Entropy > tt.wantEntropy+0.2 {
				t.Errorf("AnalyzeBinaryDiff() Entropy = %v, want approximately %v",
					stats.Entropy, tt.wantEntropy)
			}
		})
	}
}

func TestGenericBinaryHandler_RollingHash(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		window   int
		wantHash uint32
	}{
		{
			name:     "basic hash",
			data:     []byte("hello"),
			window:   4,
			wantHash: uint32(6385), // pre-calculated value
		},
		{
			name:     "empty data",
			data:     []byte{},
			window:   4,
			wantHash: 0,
		},
		{
			name:     "window larger than data",
			data:     []byte("abc"),
			window:   4,
			wantHash: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewGenericBinaryHandler()
			hash := h.rollingHash(tt.data, tt.window)
			if hash != tt.wantHash {
				t.Errorf("rollingHash() = %v, want %v", hash, tt.wantHash)
			}
		})
	}
}

func TestGenericBinaryHandler_LargeFiles(t *testing.T) {
	// Generate large test files
	oldData := generateTestData(5 * 1024 * 1024) // 5MB
	newData := modifyTestData(oldData, 1000)     // Modify every 1000th byte

	h := NewGenericBinaryHandler()
	chunks, err := h.Compare(oldData, newData)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}

	// Verify patch reconstruction
	patched, err := h.Patch(oldData, chunks)
	if err != nil {
		t.Fatalf("Patch() error = %v", err)
	}

	if !bytes.Equal(patched, newData) {
		t.Error("Large file patch failed to reconstruct original content")
	}

	// Verify stats
	stats := h.GetLatestStats()
	if stats == nil {
		t.Error("GetLatestStats() returned nil")
	}
	if stats.ChunkCount == 0 {
		t.Error("Expected non-zero chunk count for large file diff")
	}
}

// Helper functions

func generateRandomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func generateTestData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

func modifyTestData(data []byte, interval int) []byte {
	modified := make([]byte, len(data))
	copy(modified, data)
	for i := 0; i < len(modified); i += interval {
		modified[i] = modified[i] ^ 0xFF
	}
	return modified
}

// Benchmark tests

func BenchmarkGenericBinaryHandler_Compare(b *testing.B) {
	sizes := []int{
		1024,             // 1KB
		1024 * 1024,      // 1MB
		10 * 1024 * 1024, // 10MB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			old := generateTestData(size)
			new := modifyTestData(old, 1000)
			h := NewGenericBinaryHandler()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := h.Compare(old, new)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGenericBinaryHandler_Patch(b *testing.B) {
	sizes := []int{
		1024,             // 1KB
		1024 * 1024,      // 1MB
		10 * 1024 * 1024, // 10MB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			old := generateTestData(size)
			new := modifyTestData(old, 1000)
			h := NewGenericBinaryHandler()
			chunks, _ := h.Compare(old, new)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := h.Patch(old, chunks)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
