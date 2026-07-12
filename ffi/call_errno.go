//go:build !windows

package ffi

import (
	"context"
	"unsafe"

	"github.com/go-webgpu/goffi/internal/arch"
	gosyscall "github.com/go-webgpu/goffi/internal/syscall"
	"github.com/go-webgpu/goffi/types"
)

// CallFunctionErrno executes a C function call and captures the C errno value
// set by the called function.
//
// The errno is read inside the assembly trampoline immediately after the C
// function returns, before the Go runtime can migrate the goroutine to a
// different OS thread. This is the only safe window for errno capture in a
// pure-Go FFI implementation.
//
// Supported platforms: Linux, macOS, FreeBSD (both amd64 and arm64).
// On Windows, use CallFunction and check syscall.GetLastError() for Win32
// errors. Windows CRT errno is not captured by this implementation.
//
// Parameters are identical to CallFunction. The first return value is the
// captured C errno (0 if the function succeeds), and the second is any Go
// error from argument validation or dispatch.
//
// Example (calling POSIX open() and checking errno on failure):
//
//	var result int32
//	cerrno, err := ffi.CallFunctionErrno(cif, openFn,
//	    unsafe.Pointer(&result),
//	    []unsafe.Pointer{unsafe.Pointer(&pathPtr), unsafe.Pointer(&flags)})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if result == -1 {
//	    log.Printf("open failed: %v", syscall.Errno(cerrno))
//	}
func CallFunctionErrno(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) (cerrno uintptr, err error) {
	return CallFunctionErrnoContext(context.Background(), cif, fn, rvalue, avalue)
}

// CallFunctionErrnoContext is like CallFunctionErrno with context support.
// It checks the context for cancellation before executing the FFI call.
func CallFunctionErrnoContext(
	ctx context.Context,
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) (cerrno uintptr, err error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, ctxErr
	}
	if cif == nil {
		return 0, &InvalidCallInterfaceError{
			Field:  "cif",
			Reason: "must not be nil",
			Index:  -1,
		}
	}
	if fn == nil {
		return 0, &InvalidCallInterfaceError{
			Field:  "fn",
			Reason: "function pointer must not be nil",
			Index:  -1,
		}
	}

	caller, ok := arch.Registry.Caller.(arch.FunctionCallerErrno)
	if !ok {
		// Architecture does not support errno capture; fall back to plain Execute.
		return 0, executeFunction(cif, fn, rvalue, avalue)
	}

	errnoFn := gosyscall.ErrnoFnAddr()
	return caller.ExecuteErrno(cif, fn, rvalue, avalue, errnoFn)
}
