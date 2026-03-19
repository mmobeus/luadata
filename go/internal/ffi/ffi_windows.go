//go:build windows

package ffi

import (
	"fmt"
	"syscall"
)

func openLibrary(path string) (uintptr, error) {
	handle, err := syscall.LoadLibrary(path)
	if err != nil {
		return 0, fmt.Errorf("LoadLibrary: %w", err)
	}
	return uintptr(handle), nil
}
