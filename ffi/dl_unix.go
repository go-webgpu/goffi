//go:build (linux || darwin) && amd64

package ffi

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// LoadLibrary loads a shared library (dlopen)
func LoadLibrary(name string) (unsafe.Pointer, error) {
	// Convert name to C-string (null-terminated)
	cname := []byte(name + "\x00")
	handle, err := unix.Dlopen(string(cname), unix.RTLD_LAZY)
	if err != nil {
		return nil, err
	}
	return unsafe.Pointer(uintptr(handle)), nil
}

// GetSymbol retrieves a function pointer (dlsym)
func GetSymbol(handle unsafe.Pointer, name string) (unsafe.Pointer, error) {
	// Convert name to C-string (null-terminated)
	cname := []byte(name + "\x00")
	sym, err := unix.Dlsym(uintptr(handle), string(cname))
	if err != nil {
		return nil, err
	}
	return unsafe.Pointer(uintptr(sym)), nil
}
