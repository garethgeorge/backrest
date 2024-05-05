package api

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"go.uber.org/zap"
)

func NewDownloadHandler(oplog *oplog.OpLog) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path[1:]
		sep := strings.Index(p, "/")
		if sep == -1 {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		restoreID := p[:sep]
		filePath := p[sep+1:]
		opID, err := strconv.ParseInt(restoreID, 16, 64)
		if err != nil {
			http.Error(w, "invalid restore ID: "+err.Error(), http.StatusBadRequest)
			return
		}
		op, err := oplog.Get(int64(opID))
		if err != nil {
			http.Error(w, "restore not found", http.StatusNotFound)
			return
		}
		restoreOp, ok := op.Op.(*v1.Operation_OperationRestore)
		if !ok {
			http.Error(w, "restore not found", http.StatusNotFound)
			return
		}
		targetPath := restoreOp.OperationRestore.GetTarget()
		if targetPath == "" {
			http.Error(w, "restore target not found", http.StatusNotFound)
			return
		}
		fullPath := filepath.Join(targetPath, filePath)

		w.Header().Set("Content-Disposition", "attachment; filename=archive.zip")
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Transfer-Encoding", "binary")

		z := zip.NewWriter(w)
		zap.L().Info("creating zip archive", zap.String("path", fullPath))
		if err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			file, err := os.OpenFile(path, os.O_RDONLY, 0)
			if err != nil {
				zap.L().Warn("error opening file", zap.String("path", path), zap.Error(err))
				return nil
			}
			defer file.Close()
			f, err := z.Create(path[len(fullPath)+1:])
			if err != nil {
				return fmt.Errorf("add file to zip archive: %w", err)
			}
			io.Copy(f, file)
			return nil
		}); err != nil {
			zap.S().Errorf("error creating zip archive: %v", err)
			http.Error(w, "error creating zip archive", http.StatusInternalServerError)
		}
		if err := z.Close(); err != nil {
			zap.S().Errorf("error closing zip archive: %v", err)
			http.Error(w, "error closing zip archive", http.StatusInternalServerError)
		}
	})
}
