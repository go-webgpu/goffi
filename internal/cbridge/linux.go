//go:build linux && !android && cgo

package cbridge

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <errno.h>
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* value;
	const char* error;
} goffi_dl_result;

static void goffi_dlopen(const char* path, int mode, goffi_dl_result* result) {
	result->value = dlopen(path, mode);
	result->error = result->value == NULL ? dlerror() : NULL;
}

static void goffi_dlsym(uintptr_t handle, const char* name, goffi_dl_result* result) {
	dlerror();
	result->value = dlsym((void*)handle, name);
	result->error = result->value == NULL ? dlerror() : NULL;
}

static void* goffi_errno_location_address(void) {
	return (void*)&__errno_location;
}
*/
import "C"

import "unsafe"

// Dlopen loads a shared library and captures the loader error, if any.
func Dlopen(path string, mode int) (uintptr, string) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	var result C.goffi_dl_result
	C.goffi_dlopen(cpath, C.int(mode), &result)
	if result.value == nil {
		return 0, dlerrorString(result.error)
	}
	return uintptr(result.value), ""
}

// Dlsym resolves a symbol and captures the loader error, if any.
func Dlsym(handle uintptr, name string) (uintptr, string) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	var result C.goffi_dl_result
	C.goffi_dlsym(C.uintptr_t(handle), cname, &result)
	if result.value == nil {
		return 0, dlerrorString(result.error)
	}
	return uintptr(result.value), ""
}

func dlerrorString(message *C.char) string {
	if message == nil {
		return ""
	}
	return C.GoString(message)
}

// ErrnoLocationAddress returns the address of libc's errno accessor.
func ErrnoLocationAddress() uintptr {
	return uintptr(C.goffi_errno_location_address())
}
