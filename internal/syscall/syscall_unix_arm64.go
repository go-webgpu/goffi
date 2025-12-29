//go:build (linux || darwin) && arm64

// AAPCS64 ABI syscall implementation (Linux, macOS on ARM64)
// ARM64 Procedure Call Standard - identical on all Unix-like systems.
package syscall

import (
	"unsafe"
)

//go:linkname runtime_cgocall runtime.cgocall
func runtime_cgocall(fn uintptr, arg unsafe.Pointer) int32

// syscall8Args matches the layout expected by syscall8 assembly.
// ARM64 AAPCS64 uses X0-X7 (8 GPRs) and D0-D7 (8 FPRs) for arguments.
//
// Layout (offsets must match assembly):
//
//	fn:    0
//	a1-a8: 8-64   (X0-X7 arguments)
//	f1-f8: 72-128 (D0-D7 arguments)
//	r1-r2: 136-144 (X0-X1 returns)
//	fr1-fr4: 152-176 (D0-D3 float returns for HFA)
//	r8:    184 (X8 - large struct return pointer)
type syscall8Args struct {
	fn                             uintptr
	a1, a2, a3, a4, a5, a6, a7, a8 uintptr // X0-X7 (offsets 8-64)
	f1, f2, f3, f4, f5, f6, f7, f8 uintptr // D0-D7 arguments (offsets 72-128)
	r1, r2                         uintptr // X0-X1 integer returns (offsets 136-144)
	fr1, fr2, fr3, fr4             uintptr // D0-D3 float returns for HFA (offsets 152-176)
	r8                             uintptr // X8 - large struct return pointer (offset 184)
}

// syscall8 is implemented in syscall_unix_arm64.s
//
//nolint:unused // Called from assembly
func syscall8(args unsafe.Pointer)

// syscall8ABI0 is the ABI0 entry point for syscall8
var syscall8ABI0 uintptr

// Call8Float calls a C function with up to 8 integer arguments and 8 float arguments.
// This follows the AAPCS64 calling convention for ARM64.
//
// Returns:
//   - r1: X0 integer return value
//   - fret: [4]float64 containing D0-D3 for HFA returns
func Call8Float(fn uintptr, gpr [8]uintptr, fpr [8]float64, r8 uintptr) (r1 uintptr, fret [4]float64) {
	args := syscall8Args{
		fn: fn,
		a1: gpr[0], a2: gpr[1], a3: gpr[2], a4: gpr[3],
		a5: gpr[4], a6: gpr[5], a7: gpr[6], a8: gpr[7],
		// Pass floats as uintptr bit patterns
		f1: *(*uintptr)(unsafe.Pointer(&fpr[0])),
		f2: *(*uintptr)(unsafe.Pointer(&fpr[1])),
		f3: *(*uintptr)(unsafe.Pointer(&fpr[2])),
		f4: *(*uintptr)(unsafe.Pointer(&fpr[3])),
		f5: *(*uintptr)(unsafe.Pointer(&fpr[4])),
		f6: *(*uintptr)(unsafe.Pointer(&fpr[5])),
		f7: *(*uintptr)(unsafe.Pointer(&fpr[6])),
		f8: *(*uintptr)(unsafe.Pointer(&fpr[7])),
		r8: r8, // X8 for large struct returns
	}
	runtime_cgocall(syscall8ABI0, unsafe.Pointer(&args))
	r1 = args.r1
	// Return all 4 float registers for HFA support
	fret[0] = *(*float64)(unsafe.Pointer(&args.fr1))
	fret[1] = *(*float64)(unsafe.Pointer(&args.fr2))
	fret[2] = *(*float64)(unsafe.Pointer(&args.fr3))
	fret[3] = *(*float64)(unsafe.Pointer(&args.fr4))
	return
}
