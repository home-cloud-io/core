//go:build client

package client

import (
	"embed"
)

//go:embed dist/*
var Files embed.FS

const Root = "dist"
