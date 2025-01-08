package webui

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"
)

var etagCacheMu sync.Mutex
var etagCache = make(map[string]string)

func calcEtag(path string, data []byte) string {
	etagCacheMu.Lock()
	defer etagCacheMu.Unlock()
	etag, ok := etagCache[path]
	if ok {
		return etag
	}

	md5sum := md5.Sum(data)
	etag = "\"" + hex.EncodeToString(md5sum[:]) + "\""
	etagCache[path] = etag
	return etag
}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			r.URL.Path += "index.html"
		}

		f, err := content.Open(contentPrefix + r.URL.Path + ".gz")
		if err == nil {
			defer f.Close()
			w.Header().Set("Content-Encoding", "gzip")
			serveFile(f, w, r, r.URL.Path)
			return
		}

		f, err = content.Open(contentPrefix + r.URL.Path)
		if err == nil {
			defer f.Close()
			serveFile(f, w, r, r.URL.Path)
			return
		}

		http.Error(w, "Not found", http.StatusNotFound)
	})
}

func serveFile(f fs.File, w http.ResponseWriter, r *http.Request, path string) {
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("ETag", calcEtag(path, data))
	http.ServeContent(w, r, path, time.Time{}, bytes.NewReader(data))
}
