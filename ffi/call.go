package ffi

import (
	"unsafe"

	"github.com/go-webgpu/goffi/internal/arch"
	"github.com/go-webgpu/goffi/internal/hostlibc"
	"github.com/go-webgpu/goffi/internal/static"
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
	if static.Enabled {
		// A static build has no libc and no cgo runtime: runtime.cgocall would
		// abort the process with "cgocall unavailable". Fail with an error
		// instead. There is no legitimate way to obtain fn here anyway, since
		// LoadLibrary and GetSymbol also fail in this mode.
		return 0, ErrStaticBuild
	}
	if hostlibc.Missing {
		// Same reasoning as the static guard: no libc means the errno import
		// and the callee itself are unbound. LoadLibrary and GetSymbol fail
		// in this mode too, so fn cannot legitimately have come from goffi.
		return 0, ErrNoHostLibc
	}
	if arch.Registry.Caller == nil {
		return 0, types.ErrUnsupportedArchitecture
	}
	// ErrnoFnAddr returns the address of __errno_location/__error on Unix and 0
	// on Windows. The assembly trampoline's conditional (TESTQ/CBZ) skips the
	// errno capture when errnoFn is 0, so this is safe on all platforms.
	errnoFn := gosyscall.ErrnoFnAddr()
	return arch.Registry.Caller.Execute(cif, fn, rvalue, avalue, errnoFn)
}
