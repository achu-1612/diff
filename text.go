package diff

import "bytes"

// TextFileHandler is a file handler for text files.
// It implements the FileHandler interface.
type TextFileHandler struct{}

// Makesure TextFileHandler implements the FileHandler interface
var _ FileHandler = &TextFileHandler{}

// Compare compares two text files and returns the differences as a slice of DiffChunk.
func (h *TextFileHandler) Compare(old, new []byte) ([]DiffChunk, error) {
	if bytes.Equal(old, new) {
		return nil, nil
	}

	chunks := []DiffChunk{}
	oldLines := bytes.Split(old, []byte{'\n'})
	newLines := bytes.Split(new, []byte{'\n'})

	// Simple line-by-line comparison
	offset := int64(0)

	for i := 0; i < len(oldLines) && i < len(newLines); i++ {
		if !bytes.Equal(oldLines[i], newLines[i]) {
			chunks = append(chunks, DiffChunk{
				Offset:    offset,
				OldData:   oldLines[i],
				NewData:   newLines[i],
				ChunkType: "text",
			})
		}

		// +1 for newline
		offset += int64(len(oldLines[i])) + 1
	}

	return chunks, nil
}

// Patch applies the given DiffChunks to the original data and returns the patched data.
func (h *TextFileHandler) Patch(original []byte, chunks []DiffChunk) ([]byte, error) {
	if len(chunks) == 0 {
		return original, nil
	}

	result := make([]byte, 0, len(original))
	lastOffset := int64(0)

	for _, chunk := range chunks {
		// Copy unchanged data
		result = append(result, original[lastOffset:chunk.Offset]...)
		// Apply the change
		result = append(result, chunk.NewData...)

		lastOffset = chunk.Offset + int64(len(chunk.OldData))
	}

	// Copy remaining unchanged data
	if lastOffset < int64(len(original)) {
		result = append(result, original[lastOffset:]...)
	}

	return result, nil
}

// GetFileType returns the type of the file handler.
func (h *TextFileHandler) GetFileType() string {
	return "text"
}
