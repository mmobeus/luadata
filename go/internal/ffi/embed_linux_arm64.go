//go:build linux && arm64

package ffi

import _ "embed"

//go:embed lib/linux_arm64/libluadata.so
var EmbeddedLib []byte

const LibName = "libluadata.so"
