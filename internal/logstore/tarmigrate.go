package logstore

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

func MigrateTarLogsInDir(ls *LogStore, dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		zap.L().Fatal("failed to read directory", zap.String("dir", dir), zap.Error(err))
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ".tar" {
			continue
		}

		if err := MigrateTarLog(ls, filepath.Join(dir, file.Name())); err != nil {
			zap.S().Warnf("failed to migrate tar log %q: %v", file.Name(), err)
		} else {
			if err := os.Remove(filepath.Join(dir, file.Name())); err != nil {
				zap.S().Warnf("failed to remove fully migrated tar log %q: %v", file.Name(), err)
			}
		}
	}
}

func MigrateTarLog(ls *LogStore, logTar string) error {
	baseName := filepath.Base(logTar)

	f, err := os.Open(logTar)
	if err != nil {
		return fmt.Errorf("failed to open tar file: %v", err)
	}

	tarReader := tar.NewReader(f)

	var count int64
	var bytes int64
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		w, err := ls.Create(baseName+"/"+strings.TrimSuffix(header.Name, ".gz"), 0, 14*24*time.Hour)
		if err != nil {
			return fmt.Errorf("failed to create log writer: %v", err)
		}

		var r io.ReadCloser = io.NopCloser(tarReader)
		if strings.HasSuffix(header.Name, ".gz") {
			r, err = gzip.NewReader(tarReader)
			if err != nil {
				return fmt.Errorf("failed to create gzip reader: %v", err)
			}
		}

		if n, err := io.Copy(w, r); err != nil {
			return fmt.Errorf("failed to copy tar entry: %v", err)
		} else {
			bytes += n
			count++
		}

		if err := r.Close(); err != nil {
			return fmt.Errorf("failed to close tar entry reader: %v", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("failed to close log writer: %v", err)
		}
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close tar file: %v", err)
	}

	zap.L().Info("migrated tar log", zap.String("log", logTar), zap.Int64("entriesCopied", count), zap.Int64("bytesCopied", bytes))

	return nil
}
