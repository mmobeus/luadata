//go:build darwin && amd64

package ffi

import _ "embed"

//go:embed lib/darwin_amd64/libluadata.dylib
var EmbeddedLib []byte

const LibName = "libluadata.dylib"
