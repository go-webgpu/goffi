//go:build (linux || darwin) && amd64

// System V AMD64 ABI syscall implementation (Linux, macOS, FreeBSD, etc.)
// This calling convention is IDENTICAL on all Unix-like systems.
package syscall

import (
	"unsafe"
)

//go:linkname runtime_cgocall runtime.cgocall
func runtime_cgocall(fn uintptr, arg unsafe.Pointer) int32

// syscallArgs matches the layout expected by syscallN assembly.
// Supports up to 15 total arguments (6 GP registers + 9 stack slots),
// matching purego's syscall15Args layout.
//
// Layout (offsets in bytes):
//
//	fn:    0
//	a1-a15: 8-128   (6 GP registers + 9 stack slots)
//	f1-f8: 128-192  (XMM0-XMM7 as bit patterns)
//	r1:    192      (RAX return)
//	r2:    200      (RDX return, used for 9-16 byte struct returns)
type syscallArgs struct {
	fn                                                               uintptr
	a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12, a13, a14, a15 uintptr
	f1, f2, f3, f4, f5, f6, f7, f8                                   uintptr
	r1, r2                                                           uintptr
}

// syscallN is implemented in syscall_unix_amd64.s
//
//nolint:unused // Called from assembly (syscall_unix_amd64.s)
func syscallN(args unsafe.Pointer)

// syscallNABI0 is the ABI0 entry point for syscallN
var syscallNABI0 uintptr

// CallNFloat calls a C function with up to 6 integer register arguments,
// 8 SSE arguments, and 9 stack-spill arguments (15 total).
//
// gpr:       first 6 GP register values (RDI, RSI, RDX, RCX, R8, R9)
// sse:       8 SSE register values (XMM0-XMM7) as float64 bit patterns
// stackArgs: additional arguments to push onto the stack before CALL
// numStack:  how many entries in stackArgs are valid (0-9)
//
// Returns:
//   - r1: RAX integer return value
//   - r2: RDX second integer return value (9-16 byte struct returns)
//   - f1: XMM0 float return value (bit pattern)
func CallNFloat(fn uintptr, gpr [6]uintptr, sse [8]float64, stackArgs [9]uintptr, numStack int) (r1 uintptr, r2 uintptr, f1 float64) {
	args := syscallArgs{
		fn: fn,
		a1: gpr[0], a2: gpr[1], a3: gpr[2],
		a4: gpr[3], a5: gpr[4], a6: gpr[5],
		// Stack spill slots: a7-a15 map to stackArgs[0]-stackArgs[8]
		a7:  stackArgs[0],
		a8:  stackArgs[1],
		a9:  stackArgs[2],
		a10: stackArgs[3],
		a11: stackArgs[4],
		a12: stackArgs[5],
		a13: stackArgs[6],
		a14: stackArgs[7],
		a15: stackArgs[8],
		// SSE arguments as bit patterns
		f1: *(*uintptr)(unsafe.Pointer(&sse[0])),
		f2: *(*uintptr)(unsafe.Pointer(&sse[1])),
		f3: *(*uintptr)(unsafe.Pointer(&sse[2])),
		f4: *(*uintptr)(unsafe.Pointer(&sse[3])),
		f5: *(*uintptr)(unsafe.Pointer(&sse[4])),
		f6: *(*uintptr)(unsafe.Pointer(&sse[5])),
		f7: *(*uintptr)(unsafe.Pointer(&sse[6])),
		f8: *(*uintptr)(unsafe.Pointer(&sse[7])),
	}
	_ = numStack // numStack is informational; assembly always pushes all 9 slots
	runtime_cgocall(syscallNABI0, unsafe.Pointer(&args))
	r1 = args.r1
	r2 = args.r2
	f1 = *(*float64)(unsafe.Pointer(&args.f1))
	return
}
