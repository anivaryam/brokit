package extractor

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrBinaryNotFound is returned when the target binary is not found in the archive.
var ErrBinaryNotFound = errors.New("binary not found in archive")

// ExtractTarGz extracts a tar.gz archive to dst directory and returns the binary path.
func ExtractTarGz(dst, archive, binName string) (string, error) {
	f, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if filepath.Base(hdr.Name) == binName && hdr.Typeflag == tar.TypeReg {
			outPath := filepath.Join(dst, binName)
			out, err := os.Create(outPath)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return "", err
			}
			if err := out.Close(); err != nil {
				return "", err
			}
			return outPath, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrBinaryNotFound, binName)
}

// ExtractZip extracts a zip archive to dst directory and returns the binary path.
func ExtractZip(dst, archive, binName string) (string, error) {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == binName {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			outPath := filepath.Join(dst, binName)
			out, err := os.Create(outPath)
			if err != nil {
				rc.Close()
				return "", err
			}
			if _, err := io.Copy(out, rc); err != nil {
				out.Close()
				rc.Close()
				return "", err
			}
			rc.Close()
			if err := out.Close(); err != nil {
				return "", err
			}
			return outPath, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrBinaryNotFound, binName)
}
