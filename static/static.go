package static

import (
	"embed"
	_ "embed"
)

//go:embed *
var FS embed.FS