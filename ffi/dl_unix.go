//go:build linux && (amd64 || arm64) && !cgo

// Linux library loading - OUR OWN implementation (NO dependencies!)
//
// Status: ✅ FULLY WORKING
// ✅ syscall6 (internal/syscall) - Core C function calls (~30ns overhead)
// ✅ Dlopen (internal/dl) - Library loading via runtime.cgocall
// ✅ Dlsym (internal/dl) - Symbol resolution via 4-layer architecture
// ✅ Dlclose (internal/dl) - Library unloading
//
// Architecture: Four-layer approach using runtime.cgocall + JMP stubs
// See docs/dev/LINUX_FFI_IMPLEMENTATION.md for technical details.

package ffi

import (
	"fmt"
	"unsafe"

	"github.com/go-webgpu/goffi/internal/dl"
)

// RTLD constants from <dlfcn.h> for dynamic library loading.
const (
	// RTLD_NOW resolves all symbols when loading the library (recommended).
	RTLD_NOW = dl.RTLD_NOW

	// RTLD_GLOBAL makes symbols available for subsequently loaded libraries.
	RTLD_GLOBAL = dl.RTLD_GLOBAL
)

// LoadLibrary loads a shared library using dlopen.
//
// This function loads the specified shared library and returns a handle for use
// with GetSymbol. The library is loaded with RTLD_NOW|RTLD_GLOBAL flags.
//
// Parameters:
//   - name: Path to the shared library (e.g., "libm.so.6", "/usr/lib/libGL.so.1")
//
// Returns:
//   - Handle to the loaded library (use with GetSymbol and FreeLibrary)
//   - Error if loading fails
//
// Example:
//
//	handle, err := ffi.LoadLibrary("libm.so.6")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer ffi.FreeLibrary(handle)
//
// Note: Always pair LoadLibrary with FreeLibrary to prevent resource leaks.
func LoadLibrary(name string) (unsafe.Pointer, error) {
	handle, err := dl.Dlopen(name, RTLD_NOW|RTLD_GLOBAL)
	if err != nil {
		return nil, &LibraryError{
			Operation: "load",
			Name:      name,
			Err:       err,
		}
	}

	return unsafe.Pointer(handle), nil
}

// GetSymbol retrieves a function pointer from a loaded library using dlsym.
//
// This function looks up a symbol (function or variable) in the loaded library
// and returns its address for use with CallFunction.
//
// Parameters:
//   - handle: Library handle from LoadLibrary
//   - name: Name of the symbol to retrieve (e.g., "sqrt", "glClear")
//
// Returns:
//   - Function pointer (use with CallFunction)
//   - Error if symbol not found or lookup fails
//
// Example:
//
//	sqrtPtr, err := ffi.GetSymbol(handle, "sqrt")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Note: The returned pointer is only valid while the library remains loaded.
func GetSymbol(handle unsafe.Pointer, name string) (unsafe.Pointer, error) {
	fnPtr, err := dl.Dlsym(uintptr(handle), name)
	if err != nil {
		return nil, &LibraryError{
			Operation: "symbol",
			Name:      name,
			Err:       err,
		}
	}

	if fnPtr == 0 {
		return nil, &LibraryError{
			Operation: "symbol",
			Name:      name,
			Err:       fmt.Errorf("symbol not found"),
		}
	}

	return unsafe.Pointer(fnPtr), nil
}

// FreeLibrary unloads a previously loaded library using dlclose.
//
// This function decrements the reference count of the loaded library. When the
// reference count reaches zero, the library is unloaded from memory.
//
// Parameters:
//   - handle: Library handle from LoadLibrary (can be nil)
//
// Returns:
//   - nil on success
//   - Error if the library could not be unloaded
//
// Example:
//
//	handle, err := ffi.LoadLibrary("libm.so.6")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer ffi.FreeLibrary(handle)
//
// Safety:
//   - Do not use function pointers obtained from this library after FreeLibrary
//   - Always pair LoadLibrary with FreeLibrary to prevent resource leaks
//   - Safe to call with nil handle (returns nil without error)
func FreeLibrary(handle unsafe.Pointer) error {
	if handle == nil {
		return nil // Allow nil handle for convenience
	}

	err := dl.Dlclose(uintptr(handle))
	if err != nil {
		return &LibraryError{
			Operation: "free",
			Name:      "<library handle>",
			Err:       err,
		}
	}
	return nil
}
