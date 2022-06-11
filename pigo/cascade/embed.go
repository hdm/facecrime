package cascade

import (
	"embed"
)

//go:embed all:*
var CascadeFiles embed.FS
