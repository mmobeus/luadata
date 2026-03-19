// Package ffi provides the low-level bridge to the Rust shared library via purego.
package ffi

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	initOnce  sync.Once
	initError error

	pLuaDataToJSON func(input uintptr, options uintptr) uintptr
	pLuaDataFree   func(ptr uintptr)
)

func ensureInit() error {
	initOnce.Do(func() {
		initError = loadLibrary()
	})
	return initError
}

func loadLibrary() error {
	libPath, cleanup, err := extractLib()
	if err != nil {
		return fmt.Errorf("luadata: extract library: %w", err)
	}
	// Note: we intentionally don't call cleanup() — the temp file must remain
	// for the lifetime of the process since the library is loaded from it.
	_ = cleanup

	lib, err := openLibrary(libPath)
	if err != nil {
		return fmt.Errorf("luadata: open library %s: %w", libPath, err)
	}

	purego.RegisterLibFunc(&pLuaDataToJSON, lib, "LuaDataToJSON")
	purego.RegisterLibFunc(&pLuaDataFree, lib, "LuaDataFree")

	return nil
}

// extractLib writes the embedded shared library to a temp file and returns its path.
func extractLib() (string, func(), error) {
	// First check if LUADATA_LIB_PATH is set (for local dev).
	if envPath := os.Getenv("LUADATA_LIB_PATH"); envPath != "" {
		return envPath, func() {}, nil
	}

	if len(EmbeddedLib) == 0 {
		return "", nil, fmt.Errorf("no embedded library for this platform; set LUADATA_LIB_PATH or run 'make build-clib'")
	}

	tmpDir, err := os.MkdirTemp("", "luadata-*")
	if err != nil {
		return "", nil, err
	}

	tmpPath := filepath.Join(tmpDir, LibName)
	if err := os.WriteFile(tmpPath, EmbeddedLib, 0o755); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", nil, err
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return tmpPath, cleanup, nil
}

// Call invokes LuaDataToJSON via purego and returns the raw JSON envelope string.
func Call(input, options string) (string, error) {
	if err := ensureInit(); err != nil {
		return "", err
	}

	inputC := cBytes(input)
	optionsC := cBytes(options)

	resultPtr := pLuaDataToJSON(
		uintptr(unsafe.Pointer(&inputC[0])),
		uintptr(unsafe.Pointer(&optionsC[0])),
	)

	// Keep the byte slices alive until after the FFI call returns,
	// preventing the GC from collecting the backing arrays.
	runtime.KeepAlive(inputC)
	runtime.KeepAlive(optionsC)

	if resultPtr == 0 {
		return "", fmt.Errorf("luadata: LuaDataToJSON returned null")
	}

	result := goString(resultPtr)
	pLuaDataFree(resultPtr)

	return result, nil
}

// cBytes converts a Go string to a null-terminated byte slice suitable for
// passing to C. The caller must keep the returned slice alive (via
// runtime.KeepAlive) until the C function returns.
func cBytes(s string) []byte {
	b := make([]byte, len(s)+1)
	copy(b, s)
	// b[len(s)] is already 0 (Go zero-initializes slices)
	return b
}

// goString reads a null-terminated C string (UTF-8 bytes) from a pointer.
// This is correct for UTF-8 because no valid multi-byte UTF-8 sequence
// contains a zero byte.
func goString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	var length int
	for {
		b := *(*byte)(unsafe.Pointer(ptr + uintptr(length)))
		if b == 0 {
			break
		}
		length++
	}
	buf := make([]byte, length)
	for i := range length {
		buf[i] = *(*byte)(unsafe.Pointer(ptr + uintptr(i)))
	}
	return string(buf)
}
