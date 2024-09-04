package rotatinglog

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
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

var ErrFileNotFound = errors.New("file not found")
var ErrNotFound = errors.New("entry not found")
var ErrBadName = errors.New("bad name")

type RotatingLog struct {
	mu           sync.Mutex
	dir          string
	lastFile     string
	maxLogFiles  int
	now          func() time.Time
	writers      map[string]*LogWriter
	nextWriterID int64
}

func NewRotatingLog(dir string, maxLogFiles int) *RotatingLog {
	return &RotatingLog{dir: dir, maxLogFiles: maxLogFiles, writers: make(map[string]*LogWriter)}
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

func (r *RotatingLog) CreateWriter() *LogWriter {
	r.mu.Lock()
	defer r.mu.Unlock()

	randID := fmt.Sprintf(".inprogress-%d-%d", time.Now().UnixNano(), r.nextWriterID)
	r.nextWriterID++

	lw := &LogWriter{
		id: randID,
		rl: r,
	}
	r.writers[randID] = lw
	return lw
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

// Subscribe accepts a channel and sends the current log data and any updates to it.
func (r *RotatingLog) Subscribe(name string, ch chan []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if lw, ok := r.writers[name]; ok {
		lw.Subscribe(ch)
		return nil
	}

	data, err := r.Read(name)
	if err != nil {
		return err
	}

	ch <- data
	close(ch)
	return nil
}

// Read
func (r *RotatingLog) Read(name string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// check for a log writer with the name
	if lw, ok := r.writers[name]; ok {
		return lw.buffer.Bytes(), nil
	}

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

type LogWriter struct {
	id          string
	mu          sync.Mutex
	buffer      bytes.Buffer
	subscribers []chan []byte // subscribers to the log
	rl          *RotatingLog
}

func (lw *LogWriter) ID() string {
	return lw.id
}

func (lw *LogWriter) Subscribe(ch chan []byte) {
	lw.mu.Lock()
	lw.subscribers = append(lw.subscribers, ch)
	ch <- bytes.Clone(lw.buffer.Bytes())
}

func (lw *LogWriter) Unsubscribe(ch chan []byte) {
	lw.mu.Lock()
	slices.DeleteFunc(lw.subscribers, func(c chan []byte) bool {
		return c == ch
	})
	lw.mu.Unlock()
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	n, err = lw.buffer.Write(p)
	if err != nil {
		return
	}
	return n, nil
}

func (lw *LogWriter) Finalize() (string, error) {
	id, err := lw.rl.Write(lw.buffer.Bytes())
	delete(lw.rl.writers, lw.id)
	return id, err
}
