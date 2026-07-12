package ffi

import (
	"unsafe"

	"github.com/go-webgpu/goffi/internal/arch"
	gosyscall "github.com/go-webgpu/goffi/internal/syscall"
	"github.com/go-webgpu/goffi/types"
)

// executeFunction calls a function through the architecture-dependent mechanism,
// always capturing C errno inside the assembly trampoline.
func executeFunction(
	cif *types.CallInterface,
	fn unsafe.Pointer,
	rvalue unsafe.Pointer,
	avalue []unsafe.Pointer,
) (syscallErrno uintptr, err error) {
	if arch.Registry.Caller == nil {
		return 0, types.ErrUnsupportedArchitecture
	}
	// ErrnoFnAddr returns the address of __errno_location/__error on Unix and 0
	// on Windows. The assembly trampoline's conditional (TESTQ/CBZ) skips the
	// errno capture when errnoFn is 0, so this is safe on all platforms.
	errnoFn := gosyscall.ErrnoFnAddr()
	return arch.Registry.Caller.Execute(cif, fn, rvalue, avalue, errnoFn)
}
