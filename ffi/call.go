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
	// The interface call makes rvalue and avalue escape ("leaking param" in
	// -gcflags=-m), so everything reachable from them through pointer-typed
	// values is heap memory that does not move while C uses it: the contract
	// the CallFunctionContext documentation states. A change that lets the
	// compiler see through this call (devirtualization, inlining Execute)
	// must keep that escape, or the arch code would read stack addresses that
	// a stack move invalidates.
	return arch.Registry.Caller.Execute(cif, fn, rvalue, avalue, errnoFn)
}
