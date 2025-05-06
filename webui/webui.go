package webui

import (
	"net/http"

	"github.com/vearutop/statigz"
)

func Handler() http.Handler {
	return statigz.FileServer(content, statigz.FSPrefix(contentPrefix))
}
