// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build !cgo && android && arm64

package fakecgo

import "unsafe"

// x_cgo_init runs before the Go runtime has enabled runtime.cgocall.  Resolve
// the API-level marker with direct AAPCS64 calls instead of internal/dl so the
// pre-Q guard cannot recurse through an uninitialized cgocall path.
//
//go:cgo_import_dynamic goffi_android_dlopen dlopen "libdl.so"
//go:cgo_import_dynamic goffi_android_dlsym dlsym "libdl.so"
//go:cgo_import_dynamic _ _ "libdl.so"

// The assembly stubs are kept separate from the generated libc wrappers: the
// latter intentionally contain only the runtime symbols needed after startup.
//
//go:linkname _android_dlopen _android_dlopen
var _android_dlopen byte

//go:linkname _android_dlsym _android_dlsym
var _android_dlsym byte

var (
	androidDlopenABI0 = uintptr(unsafe.Pointer(&_android_dlopen))
	androidDlsymABI0  = uintptr(unsafe.Pointer(&_android_dlsym))
)

var (
	androidLibcName  = [...]byte{'l', 'i', 'b', 'c', '.', 's', 'o', 0}
	androidAPISymbol = [...]byte{
		'a', 'n', 'd', 'r', 'o', 'i', 'd', '_', 'g', 'e', 't', '_',
		'd', 'e', 'v', 'i', 'c', 'e', '_', 'a', 'p', 'i', '_', 'l',
		'e', 'v', 'e', 'l', 0,
	}
)

const androidRTLDNow = uintptr(0x00002)
const androidMinAPI = uintptr(29)

// androidAPI29 reports whether libc exposes the API-29 marker and returns a
// sufficiently new level. It uses only direct call5/assembly calls and is
// safe to invoke from x_cgo_init before ordinary runtime.cgocall exists.
//
//go:nosplit
func androidAPI29() bool {
	handle := call5(androidDlopenABI0,
		uintptr(unsafe.Pointer(&androidLibcName[0])), androidRTLDNow, 0, 0, 0)
	if handle == 0 {
		return false
	}
	apiFn := call5(androidDlsymABI0, handle,
		uintptr(unsafe.Pointer(&androidAPISymbol[0])), 0, 0, 0)
	if apiFn == 0 {
		return false
	}
	api := call5(apiFn, 0, 0, 0, 0, 0)
	// Do not close libc here. The runtime's Android path keeps libc resident,
	// and avoiding a second loader transition keeps this pre-runtime probe
	// deterministic.
	return api >= androidMinAPI
}
