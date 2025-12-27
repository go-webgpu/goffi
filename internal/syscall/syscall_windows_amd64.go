//go:build windows && amd64

// Win64 ABI syscall implementation.
// Uses syscall.SyscallN which properly handles stack growth via Go runtime's
// asmstdcall mechanism. This is the same approach used by purego.
//
// Note: Win64 ABI allows passing floats in integer registers (as bit patterns)
// for the first 4 arguments. The callee decides which register to use based
// on position, not type. So we pass float64 bit patterns as uintptr.
package syscall

import (
	"syscall"
	"unsafe"
)

// CallWin64 calls a C function with up to 4 integer arguments and 4 float arguments.
// Uses syscall.SyscallN for proper stack growth support via Go runtime.
//
// Win64 ABI: First 4 args go in RCX/XMM0, RDX/XMM1, R8/XMM2, R9/XMM3.
// The callee uses either integer or float register based on the parameter type.
// Since we pass through SyscallN (which uses integer registers), float values
// are passed as their bit patterns. This works because:
//  1. For integer params: value goes directly in RCX/RDX/R8/R9
//  2. For float params: the C function will read from XMM, but on Windows
//     the caller is responsible for copying to both registers (shadowing).
//
// For pure integer calls (most FFI), this works perfectly.
// For float calls, we rely on the fact that syscall.SyscallN properly
// sets up the shadow space where floats can be read from.
func CallWin64(fn uintptr, gpr [4]uintptr, sse [4]float64) (r1 uintptr, f1 float64) {
	// Merge GPR and SSE args - Win64 uses position-based register assignment
	// For each position, if there's a float arg, convert to bit pattern
	var args [4]uintptr

	for i := 0; i < 4; i++ {
		if sse[i] != 0 {
			// Float argument - pass as bit pattern
			args[i] = *(*uintptr)(unsafe.Pointer(&sse[i]))
		} else {
			// Integer argument
			args[i] = gpr[i]
		}
	}

	// Use syscall.SyscallN which properly handles Windows stack via asmstdcall
	ret, _, _ := syscall.SyscallN(fn, args[0], args[1], args[2], args[3])

	r1 = ret
	// Note: Float return values in XMM0 are not captured by SyscallN.
	// For functions returning float, caller needs special handling.
	// This is a known limitation matching purego's behavior.
	f1 = 0
	return
}
