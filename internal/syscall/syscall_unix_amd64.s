//go:build (linux || darwin) && amd64

#include "textflag.h"
#include "abi_amd64.h"

// syscall6 calls a C function with up to 6 integer and 8 float arguments.
// System V AMD64 ABI calling convention (identical on Linux, macOS, FreeBSD).
// This implementation closely follows purego's syscall15X pattern, adapted for syscall6.
//
// syscall6 takes a pointer to syscall6Args struct:
// struct {
//	fn    uintptr  // offset 0
//	a1    uintptr  // offset 8
//	a2    uintptr  // offset 16
//	a3    uintptr  // offset 24
//	a4    uintptr  // offset 32
//	a5    uintptr  // offset 40
//	a6    uintptr  // offset 48
//	f1    uintptr  // offset 56 (float64 as bit pattern)
//	f2    uintptr  // offset 64
//	f3    uintptr  // offset 72
//	f4    uintptr  // offset 80
//	f5    uintptr  // offset 88
//	f6    uintptr  // offset 96
//	f7    uintptr  // offset 104
//	f8    uintptr  // offset 112
//	r1    uintptr  // offset 120 (return value)
//	r2    uintptr  // offset 128 (second return register)
// }
//
// syscall6 must be called on the g0 stack with runtime.cgocall.
GLOBL ·syscall6ABI0(SB), NOPTR|RODATA, $8
DATA ·syscall6ABI0(SB)/8, $syscall6(SB)

TEXT syscall6(SB), NOSPLIT|NOFRAME, $0
	PUSHQ BP
	MOVQ  SP, BP
	SUBQ  $STACK_SIZE, SP
	MOVQ  DI, PTR_ADDRESS(BP) // save the pointer
	MOVQ  DI, R11             // R11 = args pointer

	// Load float arguments into XMM0-XMM7 (offsets 56-112)
	MOVQ 56(R11), X0  // f1
	MOVQ 64(R11), X1  // f2
	MOVQ 72(R11), X2  // f3
	MOVQ 80(R11), X3  // f4
	MOVQ 88(R11), X4  // f5
	MOVQ 96(R11), X5  // f6
	MOVQ 104(R11), X6 // f7
	MOVQ 112(R11), X7 // f8

	// Load integer arguments into registers (System V AMD64 ABI, offsets 8-48)
	MOVQ 8(R11), DI   // a1
	MOVQ 16(R11), SI  // a2
	MOVQ 24(R11), DX  // a3
	MOVQ 32(R11), CX  // a4
	MOVQ 40(R11), R8  // a5
	MOVQ 48(R11), R9  // a6

	// For vararg functions: AL = number of float args in XMM registers
	// Set to 0 to indicate "no float args" (safest for C varargs)
	XORL AX, AX

	// Load function pointer and call (offset 0)
	MOVQ 0(R11), R10 // fn
	CALL R10

	// Get the pointer back
	MOVQ PTR_ADDRESS(BP), DI

	// Save return values
	MOVQ AX, 120(DI) // r1: integer return in RAX
	MOVQ DX, 128(DI) // r2: second integer return in RDX
	MOVQ X0, 56(DI)  // f1: float return in XMM0
	MOVQ X1, 64(DI)  // f2: second float return in XMM1

	// Restore stack and return
	XORL AX, AX          // no error (ignored by runtime.cgocall)
	ADDQ $STACK_SIZE, SP
	MOVQ BP, SP
	POPQ BP
	RET
