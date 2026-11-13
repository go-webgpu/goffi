// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build linux && cgo && (amd64 || arm64) && !goffi_static

package dl

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <errno.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

typedef struct {
	void* value;
	char* error;
} goffi_dl_result;

static void goffi_capture_dlerror(goffi_dl_result* result) {
	const char* message = dlerror();
	result->error = message == NULL ? NULL : strdup(message);
}

static void goffi_cgo_dlopen(const char* path, int mode, goffi_dl_result* result) {
	result->value = dlopen(path, mode);
	result->error = NULL;
	if (result->value == NULL) {
		goffi_capture_dlerror(result);
	}
}

static void goffi_cgo_dlsym(uintptr_t handle, const char* name, goffi_dl_result* result) {
	// Clear a prior loader error before resolving this symbol.
	dlerror();
	result->value = dlsym((void*)handle, name);
	result->error = NULL;
	if (result->value == NULL) {
		goffi_capture_dlerror(result);
	}
}

static void goffi_cgo_free(char* value) {
	free(value);
}

// <errno.h> declares the accessor under any feature-test macros, so a C
// library without it fails the link instead of leaving errno capture off.
static uintptr_t goffi_errno_addr(void) {
#ifdef __ANDROID__
	return (uintptr_t)&__errno;
#else
	return (uintptr_t)&__errno_location;
#endif
}
*/
import "C"

import "fmt"

// Dlopen uses an ordinary cgo call on Linux and Android when cgo is enabled.
// A cgo binary is linked externally whenever something asks for it (-race,
// -linkmode=external, a cgo package that cannot be linked internally), and
// the external linker cannot resolve a hand-written branch to a
// cgo_import_dynamic symbol (SDYNIMPORT). The API and error semantics are
// the same as the no-cgo implementation's.
func Dlopen(path string, mode int) (uintptr, error) {
	cpath := C.CString(path)
	defer C.goffi_cgo_free(cpath)

	var result C.goffi_dl_result
	C.goffi_cgo_dlopen(cpath, C.int(mode), &result)
	if result.value == nil {
		return 0, fmt.Errorf("dlopen failed: %s", takeDlerror(result.error))
	}
	return uintptr(result.value), nil
}

// Dlsym returns the address of a symbol in a loaded library.
func Dlsym(handle uintptr, name string) (uintptr, error) {
	cname := C.CString(name)
	defer C.goffi_cgo_free(cname)

	var result C.goffi_dl_result
	C.goffi_cgo_dlsym(C.uintptr_t(handle), cname, &result)
	if result.value == nil {
		return 0, fmt.Errorf("dlsym failed: %s", takeDlerror(result.error))
	}
	return uintptr(result.value), nil
}

// Dlclose returns nil and leaves the library loaded, as the no-cgo
// implementations do: unloading is not implemented yet on Linux, and on
// Android ffi.LoadLibrary opens every library with RTLD_NODELETE.
func Dlclose(uintptr) error { return nil }

func takeDlerror(msg *C.char) string {
	if msg == nil {
		return "unknown error"
	}
	defer C.goffi_cgo_free(msg)
	return C.GoString(msg)
}

// ErrnoAddr returns the address of the errno accessor for the syscall
// package: Bionic's __errno on Android, __errno_location on glibc and musl.
// The returned address is called by syscallN immediately after the target C
// function, while execution remains on the same OS thread.
func ErrnoAddr() uintptr {
	return uintptr(C.goffi_errno_addr())
}
