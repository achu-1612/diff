package diff

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCompare(t *testing.T) {
	handler := NewGenericBinaryHandler()

	oldData, err := os.ReadFile("./testdata/bin1")
	if err != nil {
		t.Fatalf("failed to read old binary file: %v", err)
	}

	newData, err := os.ReadFile("./testdata/bin2")
	if err != nil {
		t.Fatalf("failed to read new binary file: %v", err)
	}

	chunks, err := handler.Compare(oldData, newData)
	if err != nil {
		t.Fatalf("Compare returned an error: %v", err)
	}

	if len(chunks) == 0 {
		t.Errorf("expected non-zero chunks, got %d", len(chunks))
	}

	for _, chunk := range chunks {
		if chunk.ChunkType != "binary" {
			t.Errorf("expected chunk type 'binary', got %s", chunk.ChunkType)
		}
	}

	t.Log(chunks[0].Offset)

	stats := handler.GetLatestStats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	if stats.MatchCount == 0 {
		t.Errorf("expected non-zero match count, got %d", stats.MatchCount)
	}

	if stats.TotalMatchedBytes == 0 {
		t.Errorf("expected non-zero total matched bytes, got %d", stats.TotalMatchedBytes)
	}
}
func TestPatch(t *testing.T) {
	handler := NewGenericBinaryHandler()

	originalData, err := os.ReadFile("./testdata/bin1")
	if err != nil {
		t.Fatalf("failed to read original binary file: %v", err)
	}

	modifiedData, err := os.ReadFile("./testdata/bin2")
	if err != nil {
		t.Fatalf("failed to read modified binary file: %v", err)
	}

	chunks, err := handler.Compare(originalData, modifiedData)
	if err != nil {
		t.Fatalf("Compare returned an error: %v", err)
	}

	patchedData, err := handler.Patch(originalData, chunks)
	if err != nil {
		t.Fatalf("Patch returned an error: %v", err)
	}

	if !bytes.Equal(patchedData, modifiedData) {
		t.Errorf("patched data does not match modified data")
	}
}
func TestCalculateEntropy(t *testing.T) {
	handler := NewGenericBinaryHandler()

	const textEntropy = 2.84535

	data := []byte("hello world")
	entropy := handler.calculateEntropy(data)

	if fmt.Sprintf("%.5f", entropy*8) != fmt.Sprintf("%.5f", textEntropy) {
		t.Errorf("expected entropy %f, got %f", textEntropy, entropy*8)
	}
}
func TestAnalyzeBinaryDiff(t *testing.T) {
	handler := NewGenericBinaryHandler()

	oldData, err := os.ReadFile("./testdata/bin1")
	if err != nil {
		t.Fatalf("failed to read old binary file: %v", err)
	}

	newData, err := os.ReadFile("./testdata/bin2")
	if err != nil {
		t.Fatalf("failed to read new binary file: %v", err)
	}

	stats, err := handler.AnalyzeBinaryDiff(oldData, newData)
	if err != nil {
		t.Fatalf("AnalyzeBinaryDiff returned an error: %v", err)
	}

	expectedStats := &BinaryDiffStats{
		MatchCount:        1,
		SmallestMatch:     8,
		LargestMatch:      18,
		TotalMatchedBytes: 18,
		AverageMatchSize:  18,
		CompressionRatio:  180.22222222222223,
		Entropy:           0.5085068654526307,
	}

	if diff := cmp.Diff(expectedStats, stats, cmpopts.IgnoreFields(BinaryDiffStats{}, "Entropy")); diff != "" {
		t.Errorf("unexpected stats (-want +got):\n%s", diff)
	}

	if fmt.Sprintf("%.5f", stats.Entropy) != fmt.Sprintf("%.5f", expectedStats.Entropy) {
		t.Errorf("expected entropy %.5f, got %.5f", stats.Entropy, expectedStats.Entropy)
	}
}
