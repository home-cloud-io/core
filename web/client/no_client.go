//go:build !client

package client

import "embed"

//go:embed static/*
var Files embed.FS

const Root = "static"

