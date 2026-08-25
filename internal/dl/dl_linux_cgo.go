//go:build linux && !android && cgo

package dl

import (
	"fmt"

	"github.com/go-webgpu/goffi/internal/cbridge"
)

// RTLD_* constants are shared with the !cgo path; see dl_linux_consts.go.

// Dlopen loads a shared library.
func Dlopen(path string, mode int) (uintptr, error) {
	handle, errMessage := cbridge.Dlopen(path, mode)
	if handle == 0 {
		if errMessage == "" {
			errMessage = "unknown error"
		}
		return 0, fmt.Errorf("dlopen failed: %s", errMessage)
	}
	return handle, nil
}

// Dlsym returns the address of a symbol in a loaded library.
func Dlsym(handle uintptr, name string) (uintptr, error) {
	symbol, errMessage := cbridge.Dlsym(handle, name)
	if symbol == 0 {
		if errMessage == "" {
			errMessage = "unknown error"
		}
		return 0, fmt.Errorf("dlsym failed: %s", errMessage)
	}
	return symbol, nil
}

// Dlclose intentionally retains process-lifetime mappings, matching the
// existing implementation's semantics.
func Dlclose(uintptr) error { return nil }
