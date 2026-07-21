// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build android && arm64

package ffi

import (
	"fmt"
	"unsafe"

	"github.com/go-webgpu/goffi/internal/dl"
)

// LoadLibrary loads a public Android shared library with eager, private
// symbol resolution. Bionic keeps the object mapped after dlclose because
// goffi's function pointers have process lifetime. Android support is
// arm64/API 29+ only.
func LoadLibrary(name string) (unsafe.Pointer, error) {
	handle, err := dl.Dlopen(name, dl.RTLD_NOW|dl.RTLD_LOCAL|dl.RTLD_NODELETE)
	if err != nil {
		return nil, &LibraryError{Operation: "load", Name: name, Err: err}
	}
	// Reinterpret the opaque loader value without a uintptr-to-pointer
	// conversion, which would make go vet assume a hidden Go heap pointer.
	return *(*unsafe.Pointer)(unsafe.Pointer(&handle)), nil
}

// GetSymbol retrieves a function or data pointer from an Android library.
func GetSymbol(handle unsafe.Pointer, name string) (unsafe.Pointer, error) {
	fnPtr, err := dl.Dlsym(uintptr(handle), name)
	if err != nil {
		return nil, &LibraryError{Operation: "symbol", Name: name, Err: err}
	}
	if fnPtr == 0 {
		return nil, &LibraryError{
			Operation: "symbol",
			Name:      name,
			Err:       fmt.Errorf("symbol not found"),
		}
	}
	// dlsym returns an address in native code, not a Go heap pointer. Preserve
	// its bits through the same vet-safe representation used by the Unix path.
	return *(*unsafe.Pointer)(unsafe.Pointer(&fnPtr)), nil
}

// FreeLibrary accepts a handle for API symmetry. internal/dl deliberately
// retains Android mappings for process lifetime, so this is safe to defer.
func FreeLibrary(handle unsafe.Pointer) error {
	if handle == nil {
		return nil
	}
	return dl.Dlclose(uintptr(handle))
}
