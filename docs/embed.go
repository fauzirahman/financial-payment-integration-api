package docs

import "embed"

//go:embed index.html swagger.json
var Files embed.FS
