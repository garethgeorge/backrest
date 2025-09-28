package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"go.uber.org/zap"
)

func NewDownloadHandler(oplog *oplog.OpLog, orchestrator *orchestrator.Orchestrator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path[1:]

		opID, signature, filePath, err := parseDownloadPath(p)
		if err != nil {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		if ok, err := checkDownloadURLSignature(opID, signature); err != nil || !ok {
			http.Error(w, fmt.Sprintf("invalid signature: %v", err), http.StatusForbidden)
			return
		}

		op, err := oplog.Get(int64(opID))
		if err != nil {
			http.Error(w, "restore not found", http.StatusNotFound)
			return
		}

		switch typedOp := op.Op.(type) {
		case *v1.Operation_OperationIndexSnapshot:
			handleIndexSnapshotDownload(w, r, orchestrator, op, typedOp, filePath)
		case *v1.Operation_OperationRestore:
			handleRestoreDownload(w, r, typedOp, filePath)
		default:
			http.Error(w, "restore not found", http.StatusNotFound)
		}
	})
}

func handleIndexSnapshotDownload(w http.ResponseWriter, r *http.Request, orchestrator *orchestrator.Orchestrator, op *v1.Operation, indexOp *v1.Operation_OperationIndexSnapshot, filePath string) {
	repoCfg, err := orchestrator.GetRepo(op.RepoId)
	if err != nil {
		http.Error(w, "error getting repo", http.StatusInternalServerError)
		return
	}

	if repoCfg.Guid != op.RepoGuid {
		http.Error(w, "repo GUID does not match", http.StatusNotFound)
		return
	}

	repo, err := orchestrator.GetRepoOrchestrator(op.RepoId)
	if err != nil {
		http.Error(w, "error getting repo", http.StatusInternalServerError)
		return
	}

	dumpErrCh := make(chan error, 1)
	piper, pipew := io.Pipe()

	go func() {
		dumpErrCh <- repo.Dump(r.Context(), indexOp.OperationIndexSnapshot.Snapshot.GetId(), filePath, pipew)
		pipew.Close()
	}()

	firstBytesBuffer := bytes.NewBuffer(nil)
	_, err = io.CopyN(firstBytesBuffer, piper, 32*1024)
	if err != nil && !errors.Is(err, io.EOF) {
		zap.S().Errorf("error copying snapshot: %v", err)
		http.Error(w, fmt.Sprintf("error copying snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	select {
	case dumpErr := <-dumpErrCh:
		if dumpErr != nil {
			zap.S().Errorf("error dumping snapshot: %v", dumpErr)
			http.Error(w, fmt.Sprintf("error dumping snapshot: %v", dumpErr), http.StatusInternalServerError)
			return
		}
	default:
	}

	if IsTarArchive(bytes.NewReader(firstBytesBuffer.Bytes())) && filepath.Ext(filePath) != ".tar" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v.tar", filePath))
	} else {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v", filePath))
	}
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, firstBytesBuffer); err != nil {
		zap.S().Errorf("error copying snapshot: %v", err)
		return
	}
	if _, err := io.Copy(w, piper); err != nil {
		zap.S().Errorf("error copying snapshot: %v", err)
	}
}

func handleRestoreDownload(w http.ResponseWriter, r *http.Request, op *v1.Operation_OperationRestore, filePath string) {
	targetPath := op.OperationRestore.GetTarget()
	if targetPath == "" {
		http.Error(w, "restore target not found", http.StatusNotFound)
		return
	}
	fullPath := filepath.Join(targetPath, filePath)

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=archive-%v.tar.gz", time.Now().Format("2006-01-02-15-04-05")))
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Transfer-Encoding", "binary")

	gzw, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if err != nil {
		zap.S().Errorf("error creating gzip writer: %v", err)
		http.Error(w, "error creating gzip writer", http.StatusInternalServerError)
		return
	}
	defer gzw.Close()

	if err := tarDirectory(gzw, fullPath); err != nil {
		zap.S().Errorf("error creating tar archive: %v", err)
		http.Error(w, "error creating tar archive", http.StatusInternalServerError)
	}
}

func parseDownloadPath(p string) (int64, string, string, error) {
	sep := strings.Index(p, "/")
	if sep == -1 {
		return 0, "", "", fmt.Errorf("invalid path")
	}
	restoreID := p[:sep]
	filePath := p[sep+1:]

	dash := strings.Index(restoreID, "-")
	if dash == -1 {
		return 0, "", "", fmt.Errorf("invalid restore ID")
	}
	opID, err := strconv.ParseInt(restoreID[:dash], 16, 64)
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid restore ID: %w", err)
	}
	signature := restoreID[dash+1:]
	return opID, signature, filePath, nil
}

func checkDownloadURLSignature(id int64, signature string) (bool, error) {
	wantSignatureBytes, err := signInt64(id)
	if err != nil {
		return false, err
	}
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false, err
	}
	return hmac.Equal(wantSignatureBytes, signatureBytes), nil
}

func tarDirectory(w io.Writer, dirpath string) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	return filepath.Walk(dirpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Create a new tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return fmt.Errorf("creating tar header for %s: %w", path, err)
		}

		// Update the name to be relative to the directory we're archiving
		relPath, err := filepath.Rel(dirpath, path)
		if err != nil {
			return fmt.Errorf("getting relative path for %s: %w", path, err)
		}
		header.Name = relPath

		// Write the header
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("writing tar header for %s: %w", path, err)
		}

		// Open the file
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening file %s: %w", path, err)
		}
		defer file.Close()

		// Copy the file contents
		if _, err := io.Copy(tw, file); err != nil {
			return fmt.Errorf("copying file %s to tar archive: %w", path, err)
		}

		return nil
	})
}

func IsTarArchive(r io.Reader) bool {
	if r == nil {
		return false
	}

	tr := tar.NewReader(r)
	_, err := tr.Next()
	return err == nil
}
