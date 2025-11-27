//go:build windows && amd64

// Package ffi provides Foreign Function Interface capabilities.
// This file contains Windows-specific callback implementation using syscall.NewCallback.
package ffi

import (
	"reflect"
	"sync"
	"syscall"
)

// Windows callback registry - tracks callbacks for documentation/debugging purposes
var windowsCallbacks struct {
	mu    sync.Mutex
	count int
}

// NewCallback registers a Go function as a C callback and returns a function pointer.
// On Windows, this delegates to syscall.NewCallback which properly handles Win64 ABI.
//
// Windows requirements:
//   - Function MUST have exactly one return value (uintptr-sized)
//   - All arguments must be uintptr-sized (8 bytes on amd64)
//   - Maximum ~1024 callbacks (Go runtime limit)
//
// Supported argument/return types:
//   - int, int64, uint, uint64, uintptr
//   - unsafe.Pointer, *T
//
// NOT supported on Windows:
//   - void return (must return something)
//   - int8, int16, int32, uint8, uint16, uint32, bool (not uintptr-sized)
//   - float32, float64 (use math.Float64bits/math.Float64frombits)
//
// Example:
//
//	cb := ffi.NewCallback(func(status, adapter, msg, userdata uintptr) uintptr {
//	    // Handle callback from C code
//	    return 0
//	})
func NewCallback(fn any) uintptr {
	if fn == nil {
		panic("ffi: callback function must not be nil")
	}

	val := reflect.ValueOf(fn)
	if val.Kind() != reflect.Func {
		panic("ffi: callback must be a function")
	}

	// Validate function signature for syscall.NewCallback compatibility
	typ := val.Type()

	// Check argument types - syscall.NewCallback requires uintptr-sized args only
	// On Windows 64-bit, only int/int64/uint/uint64/uintptr/pointer types work
	for i := 0; i < typ.NumIn(); i++ {
		argType := typ.In(i)
		switch argType.Kind() {
		case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64,
			reflect.Uintptr, reflect.Ptr, reflect.UnsafePointer:
			// Supported uintptr-sized types (8 bytes on amd64)
		case reflect.Float32, reflect.Float64:
			panic("ffi: float arguments not supported in Windows callbacks (use uintptr and math.Float64bits)")
		case reflect.Int8, reflect.Int16, reflect.Int32,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Bool:
			panic("ffi: Windows callbacks require uintptr-sized arguments, got " + argType.String() +
				" (use uintptr instead)")
		default:
			panic("ffi: unsupported argument type in callback: " + argType.String())
		}
	}

	// Check return type - syscall.NewCallback requires exactly one return value
	if typ.NumOut() != 1 {
		panic("ffi: Windows callbacks must have exactly one uintptr-sized return value")
	}
	{
		retType := typ.Out(0)
		switch retType.Kind() {
		case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64,
			reflect.Uintptr, reflect.Ptr, reflect.UnsafePointer:
			// Supported uintptr-sized return types (8 bytes on amd64)
		case reflect.Float32, reflect.Float64:
			panic("ffi: float return type not supported in Windows callbacks")
		case reflect.Int8, reflect.Int16, reflect.Int32,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Bool:
			panic("ffi: Windows callbacks require uintptr-sized return type, got " + retType.String())
		default:
			panic("ffi: unsupported return type in callback: " + retType.String())
		}
	}

	// Track callback count for debugging
	windowsCallbacks.mu.Lock()
	windowsCallbacks.count++
	windowsCallbacks.mu.Unlock()

	// Delegate to Go's built-in syscall.NewCallback which handles Win64 ABI correctly
	// syscall.NewCallback expects function with uintptr-sized arguments
	return syscall.NewCallback(fn)
}

// CallbackCount returns the number of callbacks registered.
// Note: On Windows, this is approximate as syscall.NewCallback manages its own registry.
func CallbackCount() int {
	windowsCallbacks.mu.Lock()
	defer windowsCallbacks.mu.Unlock()
	return windowsCallbacks.count
}
