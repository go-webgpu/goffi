//go:build linux && !android && cgo && (amd64 || arm64)

package syscall

import "github.com/go-webgpu/goffi/internal/cbridge"

// ErrnoFnAddr returns the address of glibc's errno-location function.
func ErrnoFnAddr() uintptr {
	return cbridge.ErrnoLocationAddress()
}
