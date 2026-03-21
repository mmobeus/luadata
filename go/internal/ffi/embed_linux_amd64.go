//go:build linux && amd64

package ffi

import _ "embed"

//go:embed lib/linux_amd64/libluadata.so
var EmbeddedLib []byte

const LibName = "libluadata.so"
