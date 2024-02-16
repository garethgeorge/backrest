package webui

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

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

	stat, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.ServeContent(w, r, path, stat.ModTime(), bytes.NewReader(data))
}
