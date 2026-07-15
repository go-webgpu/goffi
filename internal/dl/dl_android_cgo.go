// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build android && cgo && arm64

package dl

/*
#cgo android LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

typedef struct {
	void* value;
	char* error;
} goffi_android_dl_result;

static void goffi_android_capture_error(goffi_android_dl_result* result) {
	const char* message = dlerror();
	result->error = message == NULL ? NULL : strdup(message);
}

static void goffi_android_dlopen(const char* path, int mode, goffi_android_dl_result* result) {
	result->value = dlopen(path, mode);
	result->error = NULL;
	if (result->value == NULL) {
		goffi_android_capture_error(result);
	}
}

static void goffi_android_dlsym(uintptr_t handle, const char* name, goffi_android_dl_result* result) {
	// Clear a prior loader error before resolving this symbol.
	dlerror();
	result->value = dlsym((void*)handle, name);
	result->error = NULL;
	if (result->value == NULL) {
		goffi_android_capture_error(result);
	}
}

static void goffi_android_free(char* value) {
	free(value);
}

static uintptr_t goffi_android_errno_addr(void) {
	return (uintptr_t)dlsym(RTLD_DEFAULT, "__errno");
}
*/
import "C"

import "fmt"

// Dlopen uses an ordinary cgo call on Android when cgo is enabled. This
// avoids passing an external-linker SDYNIMPORT through a hand-written branch
// relocation while retaining the same API and error semantics.
func Dlopen(path string, mode int) (uintptr, error) {
	cpath := C.CString(path)
	defer C.goffi_android_free(cpath)

	var result C.goffi_android_dl_result
	C.goffi_android_dlopen(cpath, C.int(mode), &result)
	if result.value == nil {
		return 0, fmt.Errorf("dlopen failed: %s", takeAndroidDlerror(result.error))
	}
	return uintptr(result.value), nil
}

// Dlsym returns the address of a symbol in a loaded Android library.
func Dlsym(handle uintptr, name string) (uintptr, error) {
	cname := C.CString(name)
	defer C.goffi_android_free(cname)

	var result C.goffi_android_dl_result
	C.goffi_android_dlsym(C.uintptr_t(handle), cname, &result)
	if result.value == nil {
		return 0, fmt.Errorf("dlsym failed: %s", takeAndroidDlerror(result.error))
	}
	return uintptr(result.value), nil
}

// Dlclose intentionally retains process-lifetime mappings, matching the
// no-cgo implementation's RTLD_NODELETE policy.
func Dlclose(uintptr) error { return nil }

func takeAndroidDlerror(msg *C.char) string {
	if msg == nil {
		return "unknown error"
	}
	defer C.goffi_android_free(msg)
	return C.GoString(msg)
}

// AndroidErrnoAddr resolves Bionic's __errno accessor for the syscall package.
// The returned address is called by syscallN immediately after the target C
// function, while execution remains on the same OS thread.
func AndroidErrnoAddr() uintptr {
	return uintptr(C.goffi_android_errno_addr())
}
