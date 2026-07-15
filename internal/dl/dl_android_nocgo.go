// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build android && !cgo && arm64

package dl

import (
	"fmt"
	"structs"
	"unsafe"
)

// Keep the no-cgo path on the Go linker's dynamic-import route. The cgo path
// uses ordinary C wrappers instead; external Android linking cannot resolve
// AAPCS64 branch relocations directly to SDYNIMPORT symbols.
//
//go:cgo_import_dynamic goffi_android_dlopen dlopen "libdl.so"
//go:cgo_import_dynamic goffi_android_dlsym dlsym "libdl.so"
//go:cgo_import_dynamic goffi_android_dlerror dlerror "libdl.so"
//go:cgo_import_dynamic goffi_android_strdup strdup "libc.so"
//go:cgo_import_dynamic goffi_android_free free "libc.so"

// Force dependencies on Android's public loader and C libraries.
//go:cgo_import_dynamic _ _ "libdl.so"
//go:cgo_import_dynamic _ _ "libc.so"

//go:linkname runtime_cgocall runtime.cgocall
//go:noescape
func runtime_cgocall(fn uintptr, arg unsafe.Pointer) int32

// The loader wrappers call dlerror and strdup before returning from the same
// runtime.cgocall. dlerror state is thread-local, so querying it from a second
// cgocall could observe a different OS thread.
type androidDlopenArgs struct {
	_        structs.HostLayout
	fn       uintptr
	errorFn  uintptr
	strdupFn uintptr
	path     *byte
	mode     int
	result   uintptr
	error    uintptr
}

type androidDlsymArgs struct {
	_        structs.HostLayout
	fn       uintptr
	errorFn  uintptr
	strdupFn uintptr
	handle   uintptr
	name     *byte
	result   uintptr
	error    uintptr
}

type androidFreeArgs struct {
	_   structs.HostLayout
	fn  uintptr
	ptr uintptr
}

//go:linkname android_dlopen_stub android_dlopen_stub
var android_dlopen_stub byte

//go:linkname android_dlsym_stub android_dlsym_stub
var android_dlsym_stub byte

//go:linkname android_dlerror_stub android_dlerror_stub
var android_dlerror_stub byte

//go:linkname android_strdup_stub android_strdup_stub
var android_strdup_stub byte

//go:linkname android_free_stub android_free_stub
var android_free_stub byte

var (
	androidDlopenStubABI0  = uintptr(unsafe.Pointer(&android_dlopen_stub))
	androidDlsymStubABI0   = uintptr(unsafe.Pointer(&android_dlsym_stub))
	androidDlerrorStubABI0 = uintptr(unsafe.Pointer(&android_dlerror_stub))
	androidStrdupStubABI0  = uintptr(unsafe.Pointer(&android_strdup_stub))
	androidFreeStubABI0    = uintptr(unsafe.Pointer(&android_free_stub))
)

func androidDlopenWrapper(unsafe.Pointer)
func androidDlsymWrapper(unsafe.Pointer)
func androidFreeWrapper(unsafe.Pointer)

var (
	androidDlopenWrapperABI0 uintptr
	androidDlsymWrapperABI0  uintptr
	androidFreeWrapperABI0   uintptr
)

func Dlopen(path string, mode int) (uintptr, error) {
	pathBytes := append([]byte(path), 0)
	args := androidDlopenArgs{
		fn:       androidDlopenStubABI0,
		errorFn:  androidDlerrorStubABI0,
		strdupFn: androidStrdupStubABI0,
		path:     &pathBytes[0],
		mode:     mode,
	}
	runtime_cgocall(androidDlopenWrapperABI0, unsafe.Pointer(&args))
	if args.result == 0 {
		return 0, fmt.Errorf("dlopen failed: %s", takeAndroidDlerror(args.error))
	}
	return args.result, nil
}

func Dlsym(handle uintptr, name string) (uintptr, error) {
	nameBytes := append([]byte(name), 0)
	args := androidDlsymArgs{
		fn:       androidDlsymStubABI0,
		errorFn:  androidDlerrorStubABI0,
		strdupFn: androidStrdupStubABI0,
		handle:   handle,
		name:     &nameBytes[0],
	}
	runtime_cgocall(androidDlsymWrapperABI0, unsafe.Pointer(&args))
	if args.result == 0 {
		return 0, fmt.Errorf("dlsym failed: %s", takeAndroidDlerror(args.error))
	}
	return args.result, nil
}

// Dlclose deliberately retains mappings for process-lifetime function
// pointers, matching the RTLD_NODELETE mode used by ffi.LoadLibrary.
func Dlclose(uintptr) error { return nil }

func takeAndroidDlerror(ptr uintptr) string {
	if ptr == 0 {
		return "unknown error"
	}
	// ptr names strdup-owned native memory. Reinterpret its bits without a
	// uintptr-to-pointer conversion so vet does not mistake it for a hidden Go
	// heap pointer.
	messagePtr := *(*unsafe.Pointer)(unsafe.Pointer(&ptr))
	length := 0
	for *(*byte)(unsafe.Add(messagePtr, length)) != 0 {
		length++
	}
	message := string(unsafe.Slice((*byte)(messagePtr), length))
	args := androidFreeArgs{fn: androidFreeStubABI0, ptr: ptr}
	runtime_cgocall(androidFreeWrapperABI0, unsafe.Pointer(&args))
	return message
}
