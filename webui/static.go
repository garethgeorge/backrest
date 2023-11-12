package static

import (
	"embed"
)

//go:embed dist/*.js dist/*.css dist/*.html
var FS embed.FS