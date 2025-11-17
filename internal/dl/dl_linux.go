//go:build linux

// OUR OWN Dlopen/Dlsym implementation - NO dependencies!
// Uses runtime.cgocall approach similar to syscall6.

package dl

import (
	"fmt"
	"unsafe"
)

// RTLD constants from <dlfcn.h>
const (
	RTLD_LAZY   = 0x00001
	RTLD_NOW    = 0x00002
	RTLD_GLOBAL = 0x00100
	RTLD_LOCAL  = 0x00000
)

//go:linkname runtime_cgocall runtime.cgocall
func runtime_cgocall(fn uintptr, arg unsafe.Pointer) int32

// Assembly stubs (JMP to dynamic symbols)
// These provide a layer of indirection to avoid taking address of dynamic symbols directly
//
//go:linkname dlopen_stub dlopen_stub
var dlopen_stub byte
var dlopen_stubABI0 = uintptr(unsafe.Pointer(&dlopen_stub))

//go:linkname dlsym_stub dlsym_stub
var dlsym_stub byte
var dlsym_stubABI0 = uintptr(unsafe.Pointer(&dlsym_stub))

//go:linkname dlerror_stub dlerror_stub
var dlerror_stub byte
var dlerror_stubABI0 = uintptr(unsafe.Pointer(&dlerror_stub))

// dlopenArgs is the argument struct for dlopen_wrapper
type dlopenArgs struct {
	fn     uintptr // offset 0 - function pointer
	path   *byte   // offset 8 - C string path
	mode   int     // offset 16 - mode flags
	_pad   int     // offset 24 - padding (unused)
	result uintptr // offset 32 - return value
}

// dlsymArgs is the argument struct for dlsym_wrapper
type dlsymArgs struct {
	fn     uintptr // offset 0 - function pointer
	handle uintptr // offset 8 - library handle
	symbol *byte   // offset 16 - C string symbol name
	_pad   int     // offset 24 - padding (unused)
	result uintptr // offset 32 - return value
}

// dlerrorArgs is the argument struct for dlerror_wrapper
type dlerrorArgs struct {
	fn     uintptr // offset 0 - function pointer
	result *byte   // offset 8 - return value (char*)
}

// Wrappers (implemented in dl_wrappers_linux.s)
func dlopen_wrapper(args unsafe.Pointer)
func dlsym_wrapper(args unsafe.Pointer)
func dlerror_wrapper(args unsafe.Pointer)

var dlopen_wrapperABI0 uintptr
var dlsym_wrapperABI0 uintptr
var dlerror_wrapperABI0 uintptr

// Dlopen loads a shared library
func Dlopen(path string, mode int) (uintptr, error) {
	// Convert Go string to C string
	pathBytes := append([]byte(path), 0)

	args := dlopenArgs{
		fn:   dlopen_stubABI0,
		path: &pathBytes[0],
		mode: mode,
	}

	runtime_cgocall(dlopen_wrapperABI0, unsafe.Pointer(&args))

	if args.result == 0 {
		errMsg := dlerrorString()
		return 0, fmt.Errorf("dlopen failed: %s", errMsg)
	}

	return args.result, nil
}

// Dlsym returns the address of a symbol in a loaded library
func Dlsym(handle uintptr, name string) (uintptr, error) {
	// Convert Go string to C string
	nameBytes := append([]byte(name), 0)

	args := dlsymArgs{
		fn:     dlsym_stubABI0,
		handle: handle,
		symbol: &nameBytes[0],
	}

	runtime_cgocall(dlsym_wrapperABI0, unsafe.Pointer(&args))

	if args.result == 0 {
		errMsg := dlerrorString()
		return 0, fmt.Errorf("dlsym failed: %s", errMsg)
	}

	return args.result, nil
}

// Dlclose unloads a dynamic library
func Dlclose(handle uintptr) error {
	// Not implemented yet
	return nil
}

// dlerrorString gets error message from dlerror()
func dlerrorString() string {
	args := dlerrorArgs{
		fn: dlerror_stubABI0,
	}

	runtime_cgocall(dlerror_wrapperABI0, unsafe.Pointer(&args))

	ptr := args.result
	if ptr == nil {
		return "unknown error"
	}

	// Find string length
	var length int
	for {
		if *(*byte)(unsafe.Add(unsafe.Pointer(ptr), length)) == 0 {
			break
		}
		length++
	}

	return string(unsafe.Slice(ptr, length))
}
