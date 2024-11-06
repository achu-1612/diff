package diff

import (
	"bytes"
	"math"
)

// GenericBinaryHandler implements sophisticated binary file comparison
type GenericBinaryHandler struct {
	MinMatchLength int
	MaxGapSize     int
	ChunkSize      int64
	Stats          *BinaryDiffStats
}

// BinaryDiffStats provides statistics about binary diff operation
type BinaryDiffStats struct {
	MatchCount        int
	TotalMatchedBytes int64
	LargestMatch      int64
	SmallestMatch     int64
	AverageMatchSize  float64
	ChunkCount        int
	CompressionRatio  float64
	Entropy           float64
}

type binaryMatch struct {
	OldOffset int64
	NewOffset int64
	Length    int64
}

func NewGenericBinaryHandler() *GenericBinaryHandler {
	return &GenericBinaryHandler{
		MinMatchLength: 8,
		MaxGapSize:     1024,
		ChunkSize:      4096,
		Stats:          &BinaryDiffStats{},
	}
}

func (h *GenericBinaryHandler) Compare(old, new []byte) ([]DiffChunk, error) {
	if bytes.Equal(old, new) {
		return nil, nil
	}

	// Pre-optimization based on data characteristics
	h.OptimizeBinaryDiff(new)

	matches := h.findMatches(old, new)
	chunks := make([]DiffChunk, 0)
	var lastOldEnd, lastNewEnd int64

	for _, match := range matches {
		if match.NewOffset > lastNewEnd {
			chunks = append(chunks, DiffChunk{
				Offset:    lastOldEnd,
				OldData:   old[lastOldEnd:match.OldOffset],
				NewData:   new[lastNewEnd:match.NewOffset],
				ChunkType: "binary",
			})
		}

		lastOldEnd = match.OldOffset + match.Length
		lastNewEnd = match.NewOffset + match.Length
	}

	if lastNewEnd < int64(len(new)) {
		chunks = append(chunks, DiffChunk{
			Offset:    lastOldEnd,
			OldData:   old[lastOldEnd:],
			NewData:   new[lastNewEnd:],
			ChunkType: "binary",
		})
	}

	// Post-analysis of the diff operation
	stats, err := h.AnalyzeBinaryDiff(old, new)
	if err != nil {
		return chunks, err
	}

	stats.ChunkCount = len(chunks)
	stats.Entropy = h.calculateEntropy(new)
	h.Stats = stats

	return chunks, nil
}

func (h *GenericBinaryHandler) findMatches(old, new []byte) []binaryMatch {
	matches := make([]binaryMatch, 0)
	if len(old) == 0 || len(new) == 0 {
		return matches
	}

	hashTable := make(map[uint32][]int64)
	for i := 0; i <= len(old)-h.MinMatchLength; i += h.MinMatchLength {
		hash := h.rollingHash(old[i:], h.MinMatchLength)
		hashTable[hash] = append(hashTable[hash], int64(i))
	}

	for i := 0; i <= len(new)-h.MinMatchLength; i += h.MinMatchLength {
		hash := h.rollingHash(new[i:], h.MinMatchLength)
		if positions, ok := hashTable[hash]; ok {
			for _, pos := range positions {
				matchLen := h.extendMatch(old[pos:], new[i:])
				if matchLen >= int64(h.MinMatchLength) {
					matches = append(matches, binaryMatch{
						OldOffset: pos,
						NewOffset: int64(i),
						Length:    matchLen,
					})
					i += int(matchLen) - 1
					break
				}
			}
		}
	}

	return h.mergeAdjacentMatches(matches)
}

func (h *GenericBinaryHandler) rollingHash(data []byte, window int) uint32 {
	if len(data) < window {
		return 0
	}

	var hash uint32
	for i := 0; i < window; i++ {
		hash = (hash << 1) + uint32(data[i])
	}
	return hash
}

func (h *GenericBinaryHandler) extendMatch(old, new []byte) int64 {
	var length int64
	maxLen := int64(math.Min(float64(len(old)), float64(len(new))))

	for length < maxLen {
		if old[length] != new[length] {
			break
		}
		length++
	}
	return length
}

func (h *GenericBinaryHandler) mergeAdjacentMatches(matches []binaryMatch) []binaryMatch {
	if len(matches) < 2 {
		return matches
	}

	merged := make([]binaryMatch, 0, len(matches))
	current := matches[0]

	for i := 1; i < len(matches); i++ {
		next := matches[i]
		gapOld := next.OldOffset - (current.OldOffset + current.Length)
		gapNew := next.NewOffset - (current.NewOffset + current.Length)

		if gapOld <= int64(h.MaxGapSize) && gapNew <= int64(h.MaxGapSize) {
			// Merge the matches
			current.Length = next.NewOffset + next.Length - current.NewOffset
		} else {
			merged = append(merged, current)
			current = next
		}
	}
	merged = append(merged, current)

	return merged
}

func (h *GenericBinaryHandler) OptimizeBinaryDiff(sampleData []byte) {
	entropy := h.calculateEntropy(sampleData)
	dataSize := len(sampleData)

	// Base optimization on entropy
	switch {
	case entropy > 0.8:
		h.MinMatchLength = 16
		h.MaxGapSize = 256
		h.ChunkSize = 8192
	case entropy > 0.5:
		h.MinMatchLength = 8
		h.MaxGapSize = 1024
		h.ChunkSize = 4096
	default:
		h.MinMatchLength = 4
		h.MaxGapSize = 2048
		h.ChunkSize = 2048
	}

	// Additional size-based optimizations
	switch {
	case dataSize > 10*1024*1024: // > 10MB
		h.ChunkSize *= 4
		h.MinMatchLength += 8
	case dataSize > 1024*1024: // > 1MB
		h.ChunkSize *= 2
		h.MinMatchLength += 4
	}
}

func (h *GenericBinaryHandler) AnalyzeBinaryDiff(old, new []byte) (*BinaryDiffStats, error) {
	matches := h.findMatches(old, new)

	stats := &BinaryDiffStats{
		MatchCount:    len(matches),
		SmallestMatch: int64(h.MinMatchLength),
	}

	if len(matches) == 0 {
		stats.CompressionRatio = 1.0
		return stats, nil
	}

	var totalSize int64
	for _, match := range matches {
		totalSize += match.Length
		if match.Length > stats.LargestMatch {
			stats.LargestMatch = match.Length
		}
		if match.Length < stats.SmallestMatch {
			stats.SmallestMatch = match.Length
		}
	}

	stats.TotalMatchedBytes = totalSize
	stats.AverageMatchSize = float64(totalSize) / float64(len(matches))
	stats.CompressionRatio = float64(len(new)) / float64(totalSize)
	stats.Entropy = h.calculateEntropy(new)

	return stats, nil
}

func (h *GenericBinaryHandler) calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	freq := make(map[byte]int)
	for _, b := range data {
		freq[b]++
	}

	entropy := 0.0
	dataLen := float64(len(data))
	for _, count := range freq {
		p := float64(count) / dataLen
		entropy -= p * math.Log2(p)
	}

	return entropy / 8.0
}

func (h *GenericBinaryHandler) Patch(original []byte, chunks []DiffChunk) ([]byte, error) {
	if len(chunks) == 0 {
		return original, nil
	}

	result := make([]byte, 0, len(original))
	lastOffset := int64(0)

	for _, chunk := range chunks {
		if chunk.Offset > lastOffset {
			result = append(result, original[lastOffset:chunk.Offset]...)
		}
		result = append(result, chunk.NewData...)
		lastOffset = chunk.Offset + int64(len(chunk.OldData))
	}

	if lastOffset < int64(len(original)) {
		result = append(result, original[lastOffset:]...)
	}

	return result, nil
}

func (h *GenericBinaryHandler) GetLatestStats() *BinaryDiffStats {
	return h.Stats
}

func (h *GenericBinaryHandler) GetFileType() string {
	return "binary"
}
