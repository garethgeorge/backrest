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
	"strings"

	"go.uber.org/zap"
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
	// Download ur as a file and save it to path
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}
	hash := sha256.Sum256(body)

	if strings.HasSuffix(url, ".bz2") {
		zap.S().Infof("decompressing bz2 archive (size=%v)...", len(body))
		body, err = io.ReadAll(bzip2.NewReader(bytes.NewReader(body)))
		if err != nil {
			return "", fmt.Errorf("bz2 decompress body: %w", err)
		}
	} else if strings.HasSuffix(url, ".zip") {
		zap.S().Infof("decompressing zip archive (size=%v)...", len(body))

		archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return "", fmt.Errorf("open zip archive: %w", err)
		}

		if len(archive.File) != 1 {
			return "", fmt.Errorf("expected zip archive to contain exactly one file, got %v", len(archive.File))
		}
		f, err := archive.File[0].Open()
		if err != nil {
			return "", fmt.Errorf("open zip archive file %v: %w", archive.File[0].Name, err)
		}

		body, err = io.ReadAll(f)
		if err != nil {
			return "", fmt.Errorf("read zip archive file %v: %w", archive.File[0].Name, err)
		}
	}

	out, err := os.Create(downloadPath)
	if err != nil {
		return "", fmt.Errorf("create file %v: %w", downloadPath, err)
	}
	defer out.Close()
	if err != nil {
		return "", fmt.Errorf("create file %v: %w", downloadPath, err)
	}
	_, err = io.Copy(out, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("copy response body to file %v: %w", downloadPath, err)
	}

	return hex.EncodeToString(hash[:]), nil
}
