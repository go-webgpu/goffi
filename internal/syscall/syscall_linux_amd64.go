//go:build linux && amd64

package syscall

import (
	"unsafe"
)

//go:linkname runtime_cgocall runtime.cgocall
func runtime_cgocall(fn uintptr, arg unsafe.Pointer) int32

// syscall6Args matches the layout expected by syscall6 assembly.
// The Go assembler auto-generates offset constants (syscall6Args_fn, etc.)
// based on this struct definition.
type syscall6Args struct {
	fn                             uintptr
	a1, a2, a3, a4, a5, a6         uintptr
	f1, f2, f3, f4, f5, f6, f7, f8 uintptr
	r1, r2                         uintptr
}

// syscall6 is implemented in syscall_linux_amd64.s
//
//nolint:unused // Called from assembly (syscall_linux_amd64.s)
func syscall6(args unsafe.Pointer)

// syscall6ABI0 is the ABI0 entry point for syscall6
var syscall6ABI0 uintptr

// Call6Float calls a C function with up to 6 integer arguments and 8 float arguments.
// This implementation closely follows purego's syscall15X pattern.
func Call6Float(fn uintptr, gpr [6]uintptr, sse [8]float64) (r1 uintptr, f1 float64) {
	args := syscall6Args{
		fn: fn,
		a1: gpr[0], a2: gpr[1], a3: gpr[2],
		a4: gpr[3], a5: gpr[4], a6: gpr[5],
		// Pass floats as uintptr bit patterns
		f1: *(*uintptr)(unsafe.Pointer(&sse[0])),
		f2: *(*uintptr)(unsafe.Pointer(&sse[1])),
		f3: *(*uintptr)(unsafe.Pointer(&sse[2])),
		f4: *(*uintptr)(unsafe.Pointer(&sse[3])),
		f5: *(*uintptr)(unsafe.Pointer(&sse[4])),
		f6: *(*uintptr)(unsafe.Pointer(&sse[5])),
		f7: *(*uintptr)(unsafe.Pointer(&sse[6])),
		f8: *(*uintptr)(unsafe.Pointer(&sse[7])),
	}
	runtime_cgocall(syscall6ABI0, unsafe.Pointer(&args))
	r1 = args.r1
	f1 = *(*float64)(unsafe.Pointer(&args.f1))
	return
}
