//go:build windows && amd64

package ffi

import (
	"syscall"
	"unsafe"
)

var (
	modkernel32        = syscall.NewLazyDLL("kernel32.dll")
	procLoadLibrary    = modkernel32.NewProc("LoadLibraryW")
	procGetProcAddress = modkernel32.NewProc("GetProcAddress")
)

// LoadLibrary loads a shared library (LoadLibraryW)
func LoadLibrary(name string) (unsafe.Pointer, error) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}

	handle, _, err := procLoadLibrary.Call(uintptr(unsafe.Pointer(namePtr)))
	if handle == 0 {
		return nil, err
	}
	return unsafe.Pointer(handle), nil
}

// GetSymbol retrieves a function pointer (GetProcAddress)
func GetSymbol(handle unsafe.Pointer, name string) (unsafe.Pointer, error) {
	namePtr := unsafe.Pointer(syscall.StringBytePtr(name))
	proc, _, err := procGetProcAddress.Call(uintptr(handle), uintptr(namePtr))
	if proc == 0 {
		return nil, err
	}
	return unsafe.Pointer(proc), nil
}
