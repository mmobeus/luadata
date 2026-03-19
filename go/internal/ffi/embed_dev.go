package ffi

// On main, no shared libraries are embedded. For local development,
// set LUADATA_LIB_PATH to point to a locally-built shared library
// (e.g. via `make build-clib`).
//
// On release tags, this file is replaced by platform-specific embed
// files that include the pre-built shared libraries.

var EmbeddedLib []byte

const LibName = "libluadata.dylib"
