//go:build goffi_static && ((linux && !android) || darwin || freebsd || (android && arm64)) && (amd64 || arm64)

package ffi

import "unsafe"

// RTLD constants match the dynamic profile so call sites compile unchanged.
// They are unused under goffi_static because LoadLibrary always fails.
const (
	RTLD_NOW    = 0x00002
	RTLD_GLOBAL = 0x00100
)

// LoadLibrary always fails under -tags goffi_static.
func LoadLibrary(name string) (unsafe.Pointer, error) {
	return nil, &LibraryError{
		Operation: "load",
		Name:      name,
		Err:       ErrStaticBuild,
	}
}

// GetSymbol always fails under -tags goffi_static.
func GetSymbol(handle unsafe.Pointer, name string) (unsafe.Pointer, error) {
	return nil, &LibraryError{
		Operation: "symbol",
		Name:      name,
		Err:       ErrStaticBuild,
	}
}

// FreeLibrary is a no-op success for nil and returns ErrStaticBuild otherwise.
func FreeLibrary(handle unsafe.Pointer) error {
	if handle == nil {
		return nil
	}
	return &LibraryError{
		Operation: "free",
		Name:      "<library handle>",
		Err:       ErrStaticBuild,
	}
}
