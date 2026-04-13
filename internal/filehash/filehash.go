package filehash

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// SumHex returns lowercase hex SHA-256 of the entire file (FR-03 / NFR-03).
func SumHex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer f.Close()
	return ReaderHex(f)
}

// ReaderHex returns lowercase hex SHA-256 of r until EOF.
func ReaderHex(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("sha256: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
