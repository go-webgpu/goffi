//go:build (linux || darwin) && arm64

// AAPCS64 ABI syscall implementation (Linux, macOS on ARM64)
// ARM64 Procedure Call Standard - identical on all Unix-like systems.
package syscall

import (
	"unsafe"
)

//go:linkname runtime_cgocall runtime.cgocall
func runtime_cgocall(fn uintptr, arg unsafe.Pointer) int32

// syscallArgs matches the layout expected by syscallN assembly.
// AAPCS64 uses X0-X7 (8 GPRs) and D0-D7 (8 FPRs) for arguments.
// Args 9+ (integer) or FP overflow are placed on the stack.
//
// Layout (offsets must match assembly exactly — verified by check_arm64.go):
//
//	fn:      0
//	a1-a8:   8-64    (X0-X7 GP register arguments)
//	a9-a15:  72-120  (stack spill slots for integer args 9-15)
//	f1-f8:   128-184 (D0-D7 FP register arguments, as bit patterns)
//	r1:      192     (X0 integer return)
//	r2:      200     (X1 integer return, 9-16 byte struct returns)
//	fr1-fr4: 208-232 (D0-D3 float returns for HFA)
//	r8:      240     (X8 - large struct return pointer)
//
// NOTE: f1-f8 and fr1-fr4 are raw bit patterns. For float32 values, the
// lower 32 bits contain the float32 representation (upper 32 bits are ignored).
type syscallArgs struct {
	fn                              uintptr
	a1, a2, a3, a4, a5, a6, a7, a8 uintptr // X0-X7 (offsets 8-64)
	a9, a10, a11, a12, a13, a14, a15 uintptr // stack spill (offsets 72-120)
	f1, f2, f3, f4, f5, f6, f7, f8 uintptr // D0-D7 arguments (offsets 128-184)
	r1, r2                          uintptr // X0-X1 integer returns (offsets 192-200)
	fr1, fr2, fr3, fr4              uintptr // D0-D3 float returns for HFA (offsets 208-232)
	r8                              uintptr // X8 - large struct return pointer (offset 240)
}

// syscallN is implemented in syscall_unix_arm64.s
//
//nolint:unused // Called from assembly
func syscallN(args unsafe.Pointer)

// syscallNABI0 is the ABI0 entry point for syscallN
var syscallNABI0 uintptr

// Call8Float calls a C function with up to 8 integer arguments and 8 float arguments.
// This is the backward-compatible entry point; for stack spill use CallNFloat.
func Call8Float(fn uintptr, gpr [8]uintptr, fpr [8]uint64, r8 uintptr) (r1 uintptr, r2 uintptr, fret [4]uint64) {
	var sa [7]uintptr
	return callNFloat(fn, gpr, fpr, sa, 0, r8)
}

// CallNFloat calls a C function with up to 8 GP register arguments, 8 FP register
// arguments, 7 stack-spill slots, and an X8 sret pointer.
func CallNFloat(fn uintptr, gpr [8]uintptr, fpr [8]uint64, stackArgs [7]uintptr, numStack int, r8 uintptr) (r1 uintptr, r2 uintptr, fret [4]uint64) {
	return callNFloat(fn, gpr, fpr, stackArgs, numStack, r8)
}

func callNFloat(fn uintptr, gpr [8]uintptr, fpr [8]uint64, stackArgs [7]uintptr, numStack int, r8 uintptr) (r1 uintptr, r2 uintptr, fret [4]uint64) {
	args := syscallArgs{
		fn: fn,
		a1: gpr[0], a2: gpr[1], a3: gpr[2], a4: gpr[3],
		a5: gpr[4], a6: gpr[5], a7: gpr[6], a8: gpr[7],
		// Stack spill slots
		a9:  stackArgs[0],
		a10: stackArgs[1],
		a11: stackArgs[2],
		a12: stackArgs[3],
		a13: stackArgs[4],
		a14: stackArgs[5],
		a15: stackArgs[6],
		// FP arguments
		f1: uintptr(fpr[0]),
		f2: uintptr(fpr[1]),
		f3: uintptr(fpr[2]),
		f4: uintptr(fpr[3]),
		f5: uintptr(fpr[4]),
		f6: uintptr(fpr[5]),
		f7: uintptr(fpr[6]),
		f8: uintptr(fpr[7]),
		r8: r8, // X8 for large struct returns
	}
	_ = numStack // informational; assembly always pushes all 7 stack slots
	runtime_cgocall(syscallNABI0, unsafe.Pointer(&args))
	r1 = args.r1
	r2 = args.r2
	fret[0] = uint64(args.fr1)
	fret[1] = uint64(args.fr2)
	fret[2] = uint64(args.fr3)
	fret[3] = uint64(args.fr4)
	return
}
