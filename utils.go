package diff

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// calculateHash calculates the SHA256 hash of a file.
func calculateHash(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}

	defer file.Close()

	hash := sha256.New()

	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// compressData compresses data using gzip.
func compressData(data []byte, compress bool, level int) []byte {
	if !compress {
		return data
	}

	var buf bytes.Buffer

	writer, _ := gzip.NewWriterLevel(&buf, level)

	writer.Write(data)
	writer.Close()

	return buf.Bytes()
}

// decompressData decompresses data using gzip.
func decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	return io.ReadAll(reader)
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}

	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
