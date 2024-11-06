package diff

import (
	"bytes"
	"compress/gzip"
	"os"
	"testing"
)

const (
	testDatadir     = "testdata"
	testFileContent = "Hello World !!!"
	testFileSHA256  = "f4cb61e283ba642738c065767c090da4c1a363f5d4cfb90cacc5167ecf13c760"
	testFileName    = "content.txt"
	testStringData  = "vzq4LuTQzzcyLEs1NYhhv3o2KGF9cVTQFTVbxoT67W5tCW3crVSVIeaMdo9Jqa2LT89e2LobT2dxm33G4VIcefy2e1Pweo3x0rTYGBL4NefwQe4dL4yw41RkLqylsB4WQ8DW4APDs20iZQ658bxQwSSl9p4tGJ09U9aUOcjW0SJRZbh6MKVwN20eRtzwKkHRgRqRI9Cy"
)

func creatTestFile(t *testing.T) {
	t.Helper()

	if err := os.MkdirAll(testDatadir, os.ModePerm); err != nil {
		t.Fatalf("Failed to create test data directory: %v", err)
	}

	if err := os.WriteFile(testDatadir+"/"+testFileName, []byte(testFileContent), os.ModePerm); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func cleanTestDir(t *testing.T) {
	t.Helper()

	if err := os.RemoveAll(testDatadir); err != nil {
		t.Fatalf("Failed to remove test data directory: %v", err)
	}
}

func Test_calculateHash(t *testing.T) {
	creatTestFile(t)
	defer cleanTestDir(t)

	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			name:     "Valid file",
			filePath: testDatadir + "/" + testFileName,
			want:     testFileSHA256,
		},
		{
			name:     "Non-existent file",
			filePath: testDatadir + "/" + "non_existent_file.txt",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateHash(tt.filePath); got != tt.want {
				t.Errorf("calculateHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_compressData(t *testing.T) {
	testData := []byte(testStringData)
	tests := []struct {
		name      string
		data      []byte
		compress  bool
		level     int
		wantError bool
	}{
		{
			name:      "No compression",
			data:      testData,
			compress:  false,
			level:     gzip.DefaultCompression,
			wantError: false,
		},
		{
			name:      "Default compression",
			data:      testData,
			compress:  true,
			level:     gzip.DefaultCompression,
			wantError: false,
		},
		{
			name:      "Best compression",
			data:      testData,
			compress:  true,
			level:     gzip.BestCompression,
			wantError: false,
		},
		{
			name:      "Best speed",
			data:      testData,
			compress:  true,
			level:     gzip.BestSpeed,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compressData(tt.data, tt.compress, tt.level)

			if tt.compress {
				if len(got) >= len(tt.data) {
					t.Errorf("compressData() = %v, want compressed data smaller than original", got)
				}

				decompressed, err := decompressData(got)
				if (err != nil) != tt.wantError {
					t.Fatalf("decompressData() error = %v, wantError %v", err, tt.wantError)
				}

				if !bytes.Equal(decompressed, tt.data) {
					t.Errorf("decompressData() = %v, want %v", decompressed, tt.data)
				}
			} else {
				if !bytes.Equal(got, tt.data) {
					t.Errorf("compressData() = %v, want %v", got, tt.data)
				}
			}
		})
	}
}
func Test_decompressData(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		want      []byte
		wantError bool
	}{
		{
			name:      "Valid compressed data",
			data:      compressData([]byte("test data"), true, gzip.DefaultCompression),
			want:      []byte("test data"),
			wantError: false,
		},
		{
			name:      "Invalid compressed data",
			data:      []byte("invalid compressed data"),
			want:      nil,
			wantError: true,
		},
		{
			name:      "Empty compressed data",
			data:      compressData([]byte(""), true, gzip.DefaultCompression),
			want:      []byte(""),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decompressData(tt.data)
			if (err != nil) != tt.wantError {
				t.Fatalf("decompressData() error = %v, wantError %v", err, tt.wantError)
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("decompressData() = %v, want %v", got, tt.want)
			}
		})
	}
}
func Test_copyFile(t *testing.T) {
	creatTestFile(t)
	defer cleanTestDir(t)

	tests := []struct {
		name      string
		src       string
		dst       string
		wantError bool
	}{
		{
			name:      "Valid file copy",
			src:       testDatadir + "/" + testFileName,
			dst:       testDatadir + "/" + "destination.txt",
			wantError: false,
		},
		{
			name:      "Source file does not exist",
			src:       testDatadir + "/" + "non_existent_file.txt",
			dst:       testDatadir + "/non_existent_copy_file.txt",
			wantError: true,
		},
		{
			name:      "Destination path is invalid",
			src:       testDatadir + "/" + testFileName,
			dst:       testDatadir + "/invalid/non_existent_copy_file.txt",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := copyFile(tt.src, tt.dst)
			if (err != nil) != tt.wantError {
				t.Fatalf("copyFile() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError {
				srcData, err := os.ReadFile(tt.src)
				if err != nil {
					t.Fatalf("Failed to read source file: %v", err)
				}

				dstData, err := os.ReadFile(tt.dst)
				if err != nil {
					t.Fatalf("Failed to read destination file: %v", err)
				}

				if !bytes.Equal(srcData, dstData) {
					t.Errorf("copyFile() = %v, want %v", dstData, srcData)
				}

				// Clean up
				os.Remove(tt.dst)
			}
		})
	}
}
