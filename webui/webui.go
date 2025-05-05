package webui

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"io"
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
			// Check if gzip encoding is supported
			var fr io.Reader = f
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				gzr, err := gzip.NewReader(f)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				defer gzr.Close()
				fr = gzr
			} else {
				w.Header().Set("Content-Encoding", "gzip")
			}
			serveFile(fr, w, r, r.URL.Path)
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

func serveFile(f io.Reader, w http.ResponseWriter, r *http.Request, path string) {
	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("ETag", calcEtag(path, data))
	http.ServeContent(w, r, path, time.Time{}, bytes.NewReader(data))
}
