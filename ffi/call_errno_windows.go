//go:build windows

package ffi

import (
	"context"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// CallFunctionErrno on Windows calls the function via CallFunction and returns
// errno 0. Windows Win32 APIs use GetLastError() for error reporting; the CRT
// errno is rarely used. Use the third return value of syscall.SyscallN (the
// last-error value) when you need Windows error codes.
func CallFunctionErrno(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) (cerrno uintptr, err error) {
	return 0, CallFunction(cif, fn, rvalue, avalue)
}

// CallFunctionErrnoContext on Windows calls the function via
// CallFunctionContext and returns errno 0.
func CallFunctionErrnoContext(
	ctx context.Context,
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) (cerrno uintptr, err error) {
	return 0, CallFunctionContext(ctx, cif, fn, rvalue, avalue)
}
