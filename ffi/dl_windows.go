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
	procFreeLibrary    = modkernel32.NewProc("FreeLibrary")
)

// LoadLibrary loads a shared library using Windows LoadLibraryW API.
//
// Parameters:
//   - name: Path to the DLL file (e.g., "kernel32.dll", "C:\path\to\lib.dll")
//
// Returns:
//   - Handle to the loaded library (use with GetSymbol and FreeLibrary)
//   - Error if loading fails
//
// Example:
//
//	handle, err := ffi.LoadLibrary("user32.dll")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer ffi.FreeLibrary(handle)
func LoadLibrary(name string) (unsafe.Pointer, error) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, &LibraryError{
			Operation: "load",
			Name:      name,
			Err:       err,
		}
	}

	handle, _, err := procLoadLibrary.Call(uintptr(unsafe.Pointer(namePtr)))
	if handle == 0 {
		return nil, &LibraryError{
			Operation: "load",
			Name:      name,
			Err:       err,
		}
	}

	// Safe: handle is immediately converted from uintptr to unsafe.Pointer
	// without intermediate storage, keeping it reachable by GC.
	return unsafe.Pointer(handle), nil
}

// GetSymbol retrieves a function pointer from a loaded library using GetProcAddress.
//
// Parameters:
//   - handle: Library handle from LoadLibrary
//   - name: Name of the function to retrieve (e.g., "MessageBoxW")
//
// Returns:
//   - Function pointer (use with CallFunction)
//   - Error if symbol not found
//
// Example:
//
//	msgBoxPtr, err := ffi.GetSymbol(handle, "MessageBoxW")
//	if err != nil {
//	    log.Fatal(err)
//	}
func GetSymbol(handle unsafe.Pointer, name string) (unsafe.Pointer, error) {
	namePtr := unsafe.Pointer(syscall.StringBytePtr(name))
	proc, _, err := procGetProcAddress.Call(uintptr(handle), uintptr(namePtr))
	if proc == 0 {
		return nil, &LibraryError{
			Operation: "symbol",
			Name:      name,
			Err:       err,
		}
	}

	// Safe: proc is immediately converted from uintptr to unsafe.Pointer
	// without intermediate storage, keeping it reachable by GC.
	return unsafe.Pointer(proc), nil
}

// FreeLibrary unloads a previously loaded library using FreeLibrary.
//
// This function decrements the reference count of the loaded DLL. When the reference
// count reaches zero, the module is unloaded from the address space of the calling process.
//
// Parameters:
//   - handle: Library handle from LoadLibrary (must not be nil)
//
// Returns:
//   - nil on success
//   - Error if the library could not be freed
//
// Example:
//
//	handle, err := ffi.LoadLibrary("user32.dll")
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

	ret, _, err := procFreeLibrary.Call(uintptr(handle))
	if ret == 0 {
		return &LibraryError{
			Operation: "free",
			Name:      "<library handle>",
			Err:       err,
		}
	}
	return nil
}
