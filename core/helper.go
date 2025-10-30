// Package core provides file similarity detection and folder comparison functionality.
// It includes storage management, file hashing, and duplicate detection algorithms.
package core

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"

	"github.com/kalafut/imohash"
)

// FileHash computes a hash for the given file using the provided hash algorithm.
func getFileHash(file fs.File, hash imohash.ImoHash) (string, error) {
	readerAt, ok := file.(io.ReaderAt)
	if !ok {
		return "", fmt.Errorf("file does not implement io.ReaderAt")
	}

	fi, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file size: %w", err)
	}

	hashValue, err := hash.SumSectionReader(io.NewSectionReader(readerAt, 0, fi.Size()))
	if err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return base64.RawStdEncoding.EncodeToString(hashValue[:]), nil

}

// FormatFileSize formats a file size in bytes into a human-readable string.
func FormatFileSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	s := float64(size)
	i := 0
	for s >= 1024 && i < len(units)-1 {
		s /= 1024
		i++
	}
	return fmt.Sprintf("%.2f%s", s, units[i])
}
