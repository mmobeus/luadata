//go:build darwin && arm64

package ffi

import _ "embed"

//go:embed lib/darwin_arm64/libluadata.dylib
var EmbeddedLib []byte

const LibName = "libluadata.dylib"
