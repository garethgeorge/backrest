package resticinstaller

import (
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// getURL downloads the given url and returns the response body as a string.
func getURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http GET %v: %w", url, err)
	}
	defer resp.Body.Close()

	var body bytes.Buffer
	_, err = io.Copy(&body, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("copy response body to buffer: %w", err)
	}
	return body.Bytes(), nil
}

// downloadFile downloads a file from the given url and saves it to the given path. The sha256 checksum of the file is returned on success.
func downloadFile(url string, downloadPath string) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http GET %v: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http GET %v: %v", url, resp.Status)
	}

	var dst *os.File
	if err := os.MkdirAll(filepath.Dir(downloadPath), 0755); err != nil {
		return "", fmt.Errorf("create directory %v: %w", filepath.Dir(downloadPath), err)
	}
	dst, err = os.Create(downloadPath)
	if err != nil {
		return "", fmt.Errorf("create file %v: %w", downloadPath, err)
	}
	defer dst.Close()

	shasum := sha256.New()
	reader := io.TeeReader(resp.Body, shasum)

	if strings.HasSuffix(url, ".bz2") {
		bz2Reader := bzip2.NewReader(reader)
		if _, err := io.Copy(dst, bz2Reader); err != nil {
			return "", fmt.Errorf("copy bz2 response body to file %v: %w", downloadPath, err)
		}
	} else if strings.HasSuffix(url, ".zip") {
		var bodyBytes []byte
		if bodyBytes, err = io.ReadAll(reader); err != nil {
			return "", fmt.Errorf("read response body: %w", err)
		}
		zipReader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
		if err != nil {
			return "", fmt.Errorf("read zip archive: %w", err)
		}
		if len(zipReader.File) != 1 {
			return "", fmt.Errorf("expected zip archive to contain exactly one file, got %v", len(zipReader.File))
		}
		f, err := zipReader.File[0].Open()
		if err != nil {
			return "", fmt.Errorf("open zip archive file %v: %w", zipReader.File[0].Name, err)
		}
		if _, err := io.Copy(dst, f); err != nil {
			return "", fmt.Errorf("copy zip archive file %v to file %v: %w", zipReader.File[0].Name, downloadPath, err)
		}
		f.Close()
	} else {
		if _, err := io.Copy(dst, reader); err != nil {
			return "", fmt.Errorf("copy response body to file %v: %w", downloadPath, err)
		}
	}
	hash := shasum.Sum(nil)

	return hex.EncodeToString(hash[:]), nil
}
