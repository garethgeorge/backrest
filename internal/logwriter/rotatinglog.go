package logwriter

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

type RotatingLog struct {
	mu          sync.Mutex
	dir         string
	lastFile    string
	maxLogFiles int
	now         func() time.Time
}

func NewRotatingLog(dir string, maxLogFiles int) *RotatingLog {
	return &RotatingLog{dir: dir, maxLogFiles: maxLogFiles}
}

func (r *RotatingLog) curfile() string {
	t := time.Now()
	if r.now != nil {
		t = r.now() // for testing
	}
	return path.Join(r.dir, t.Format("2006-01-02-logs.tar"))
}

func (r *RotatingLog) removeExpiredFiles() error {
	if r.maxLogFiles < 0 {
		return nil
	}
	files, err := r.files()
	if err != nil {
		return fmt.Errorf("list files: %w", err)
	}
	if len(files) >= r.maxLogFiles {
		for i := 0; i < len(files)-r.maxLogFiles+1; i++ {
			if err := os.Remove(path.Join(r.dir, files[i])); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *RotatingLog) Write(data []byte) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := compress(data)
	if err != nil {
		return "", err
	}

	file := r.curfile()
	if file != r.lastFile {
		if err := os.MkdirAll(r.dir, os.ModePerm); err != nil {
			return "", err
		}
		r.lastFile = file
		if err := r.removeExpiredFiles(); err != nil {
			zap.L().Error("failed to remove expired files for rotatinglog", zap.Error(err), zap.String("dir", r.dir))
		}
	}
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer f.Close()

	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return "", err
	}
	pos := int64(0)
	if size != 0 {
		pos, err = f.Seek(-1024, io.SeekEnd)
		if err != nil {
			return "", err
		}
	}
	tw := tar.NewWriter(f)
	defer tw.Close()

	tw.WriteHeader(&tar.Header{
		Name:     fmt.Sprintf("%d.gz", pos),
		Size:     int64(len(data)),
		Mode:     0600,
		Typeflag: tar.TypeReg,
		ModTime:  time.Now(),
	})

	_, err = tw.Write(data)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%d", path.Base(file), pos), nil
}

func (r *RotatingLog) Read(name string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// parse name e.g. of the form "2006-01-02-15-04-05.tar/1234"
	splitAt := strings.Index(name, "/")
	if splitAt == -1 {
		return nil, ErrBadName
	}

	offset, err := strconv.Atoi(name[splitAt+1:])
	if err != nil {
		return nil, ErrBadName
	}

	// open file and seek to the offset where the tarball segment should start
	f, err := os.Open(path.Join(r.dir, name[:splitAt]))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("open failed: %w", err)
	}
	defer f.Close()
	f.Seek(int64(offset), io.SeekStart)

	// search for the tarball segment in the tarball and read + decompress it if found
	seekName := fmt.Sprintf("%d.gz", offset)
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("next failed: %v", err)
		}
		if hdr.Name == seekName {
			buf := make([]byte, hdr.Size)
			_, err := io.ReadFull(tr, buf)
			if err != nil {
				return nil, fmt.Errorf("read failed: %v", err)
			}
			return decompress(buf)
		}
	}
	return nil, ErrNotFound
}

func (r *RotatingLog) files() ([]string, error) {
	files, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}
	files = slices.DeleteFunc(files, func(f fs.DirEntry) bool {
		return f.IsDir() || !strings.HasSuffix(f.Name(), "-logs.tar")
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})
	var result []string
	for _, f := range files {
		result = append(result, f.Name())
	}
	return result, nil
}

func compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	if _, err := zw.Write(data); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decompress(compressedData []byte) ([]byte, error) {
	var buf bytes.Buffer
	zr, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(&buf, zr); err != nil {
		return nil, err
	}

	if err := zr.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
