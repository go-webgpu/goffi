//go:build (linux || darwin) && amd64

package ffi

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// LoadLibrary loads a shared library (dlopen).
func LoadLibrary(name string) (unsafe.Pointer, error) {
	nameC := append([]byte(name), 0)
	handle, err := unix.Dlopen(string(nameC), unix.RTLD_LAZY)
	if err != nil {
		return nil, err
	}
	return unsafe.Pointer(uintptr(handle)), nil
}

// GetSymbol retrieves a function pointer (dlsym).
func GetSymbol(handle, name unsafe.Pointer) (unsafe.Pointer, error) {
	nameStr := *(*string)(name)
	nameC := append([]byte(nameStr), 0)
	sym, err := unix.Dlsym(uintptr(handle), string(nameC))
	if err != nil {
		return nil, err
	}
	return unsafe.Pointer(uintptr(sym)), nil
}
