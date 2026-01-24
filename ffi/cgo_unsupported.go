//go:build (linux || darwin) && cgo

// Package ffi cannot be built with CGO_ENABLED=1.
//
// goffi is a pure Go FFI library that uses Go's cgo_import_dynamic mechanism
// for dynamic library loading. This mechanism only works when CGO is disabled.
//
// To fix this error, build with CGO disabled:
//
//	CGO_ENABLED=0 go build ./...
//
// Or set permanently:
//
//	go env -w CGO_ENABLED=0
//
// For cross-compilation, CGO is automatically disabled:
//
//	GOOS=linux GOARCH=arm64 go build ./...
//
// For more information, see:
//
//	https://github.com/go-webgpu/goffi#requirements

package ffi

// Compile-time assertion: build fails immediately with descriptive error.
// The identifier name explains the issue; see package documentation above for details.
var _ = GOFFI_REQUIRES_CGO_ENABLED_0

// Runtime fallback: provides detailed instructions if compile-time check is bypassed.
// This should never execute under normal circumstances.
func init() {
	panic(`
================================================================================
goffi: CGO_ENABLED=1 is not supported
================================================================================

goffi is a pure Go FFI library that requires CGO_ENABLED=0 to build.

This error occurs because:
  - You have a C compiler (gcc/clang) installed
  - Go automatically enables CGO when a C compiler is available
  - goffi uses Go's cgo_import_dynamic mechanism which only works with CGO_ENABLED=0

To fix this error, build with CGO disabled:

  Option 1: Set for single build
    CGO_ENABLED=0 go build ./...

  Option 2: Set for current shell session
    export CGO_ENABLED=0
    go build ./...

  Option 3: Set permanently via go env
    go env -w CGO_ENABLED=0

For cross-compilation, CGO is automatically disabled:
    GOOS=linux GOARCH=arm64 go build ./...

For more information:
  https://github.com/go-webgpu/goffi#requirements

================================================================================
`)
}
