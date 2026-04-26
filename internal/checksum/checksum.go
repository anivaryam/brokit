package checksum

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrMismatch is returned when the checksum doesn't match.
type ErrMismatch struct {
	Expected string
	Actual   string
}

func (e *ErrMismatch) Error() string {
	return fmt.Sprintf("checksum mismatch: expected %s, got %s", e.Expected, e.Actual)
}

// SHA256 downloads a file and returns its SHA256 hash.
func SHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SHA256FromHexFile reads a checksums file (like SHA256SUMS) and returns the
// expected hash for a given filename. The file format is expected to be:
//   <hash> <filename>
//   <hash> <filename>
//
// For example:
//   a1b2c3d4e5f6...  mybinary
//   f6e5d4c3b2a1...  otherbinary
func SHA256FromHexFile(checksumFile, filename string) (string, error) {
	f, err := os.Open(checksumFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Format: <hash> <filename>
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		hash := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		// Support both full path and basename
		if name == filename || strings.HasSuffix(name, "/"+filename) {
			return hash, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("checksum for %s not found in %s", filename, checksumFile)
}

// VerifyFile checks that the file at path matches the expected SHA256 hash.
func VerifyFile(path, expected string) error {
	actual, err := SHA256(path)
	if err != nil {
		return err
	}
	if actual != expected {
		return &ErrMismatch{Expected: expected, Actual: actual}
	}
	return nil
}