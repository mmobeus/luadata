//go:build windows && amd64

package ffi

import _ "embed"

//go:embed lib/windows_amd64/luadata.dll
var EmbeddedLib []byte

const LibName = "luadata.dll"
