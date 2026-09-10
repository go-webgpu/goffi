//go:build goffi_static

// Package fakecgo is a no-op under -tags goffi_static.
//
// The dynamic-profile fakecgo implementation pulls libc/libpthread via
// //go:cgo_import_dynamic, which forces PT_INTERP + DT_NEEDED even when
// CGO_ENABLED=0. Static builds exclude that code entirely: FFI LoadLibrary is
// stubbed and runtime.cgocall is not required for a pure-Go static binary.
package fakecgo
