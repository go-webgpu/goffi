//go:build (linux || darwin) && amd64

#include "textflag.h"
#include "abi_amd64.h"

// syscallN calls a C function with up to 6 integer, 8 float, and 9 stack arguments.
// System V AMD64 ABI calling convention (identical on Linux, macOS, FreeBSD).
// This implementation follows purego's syscall15X pattern.
//
// syscallN takes a pointer to syscallArgs struct:
// struct {
//	fn    uintptr  // offset 0
//	a1    uintptr  // offset 8   (RDI)
//	a2    uintptr  // offset 16  (RSI)
//	a3    uintptr  // offset 24  (RDX)
//	a4    uintptr  // offset 32  (RCX)
//	a5    uintptr  // offset 40  (R8)
//	a6    uintptr  // offset 48  (R9)
//	a7    uintptr  // offset 56  (stack[0])
//	a8    uintptr  // offset 64  (stack[1])
//	a9    uintptr  // offset 72  (stack[2])
//	a10   uintptr  // offset 80  (stack[3])
//	a11   uintptr  // offset 88  (stack[4])
//	a12   uintptr  // offset 96  (stack[5])
//	a13   uintptr  // offset 104 (stack[6])
//	a14   uintptr  // offset 112 (stack[7])
//	a15   uintptr  // offset 120 (stack[8])
//	f1    uintptr  // offset 128 (XMM0 bit pattern)
//	f2    uintptr  // offset 136 (XMM1)
//	f3    uintptr  // offset 144 (XMM2)
//	f4    uintptr  // offset 152 (XMM3)
//	f5    uintptr  // offset 160 (XMM4)
//	f6    uintptr  // offset 168 (XMM5)
//	f7    uintptr  // offset 176 (XMM6)
//	f8    uintptr  // offset 184 (XMM7)
//	r1    uintptr  // offset 192 (RAX return)
//	r2    uintptr  // offset 200 (RDX return, 9-16 byte struct)
// }
//
// syscallN must be called on the g0 stack with runtime.cgocall.
//
// Stack frame layout (STACK_SIZE = 80):
//   SP+0  .. SP+71 : 9 stack-spill slots for a7-a15 (9 * 8 = 72 bytes)
//   SP+72 .. SP+79 : saved args pointer (PTR_ADDRESS = 72)
//   [then BP push/pop outside STACK_SIZE]
GLOBL ·syscallNABI0(SB), NOPTR|RODATA, $8
DATA ·syscallNABI0(SB)/8, $syscallN(SB)

TEXT syscallN(SB), NOSPLIT|NOFRAME, $0
	PUSHQ BP
	MOVQ  SP, BP
	SUBQ  $STACK_SIZE, SP
	MOVQ  DI, PTR_ADDRESS(BP) // save the pointer
	MOVQ  DI, R11             // R11 = args pointer

	// Load float arguments into XMM0-XMM7 (offsets 128-184)
	MOVQ 128(R11), X0 // f1
	MOVQ 136(R11), X1 // f2
	MOVQ 144(R11), X2 // f3
	MOVQ 152(R11), X3 // f4
	MOVQ 160(R11), X4 // f5
	MOVQ 168(R11), X5 // f6
	MOVQ 176(R11), X6 // f7
	MOVQ 184(R11), X7 // f8

	// Push stack-spill arguments a7-a15 onto the stack (offsets 56-120)
	// System V AMD64 ABI: args 7+ are pushed right-to-left onto the stack,
	// but since we build the stack growing downward with explicit offsets,
	// we push a7 at SP+0, a8 at SP+8, etc.
	MOVQ 56(R11), R12
	MOVQ R12, 0(SP)   // push a7
	MOVQ 64(R11), R12
	MOVQ R12, 8(SP)   // push a8
	MOVQ 72(R11), R12
	MOVQ R12, 16(SP)  // push a9
	MOVQ 80(R11), R12
	MOVQ R12, 24(SP)  // push a10
	MOVQ 88(R11), R12
	MOVQ R12, 32(SP)  // push a11
	MOVQ 96(R11), R12
	MOVQ R12, 40(SP)  // push a12
	MOVQ 104(R11), R12
	MOVQ R12, 48(SP)  // push a13
	MOVQ 112(R11), R12
	MOVQ R12, 56(SP)  // push a14
	MOVQ 120(R11), R12
	MOVQ R12, 64(SP)  // push a15

	// Load integer arguments into GP registers (System V AMD64 ABI, offsets 8-48)
	MOVQ 8(R11), DI   // a1 -> RDI
	MOVQ 16(R11), SI  // a2 -> RSI
	MOVQ 24(R11), DX  // a3 -> RDX
	MOVQ 32(R11), CX  // a4 -> RCX
	MOVQ 40(R11), R8  // a5 -> R8
	MOVQ 48(R11), R9  // a6 -> R9

	// For vararg functions: AL = number of float args in XMM registers (set 0 = safe default)
	XORL AX, AX

	// Load function pointer and call (offset 0)
	MOVQ 0(R11), R10
	CALL R10

	// Restore pointer and save return values
	MOVQ PTR_ADDRESS(BP), DI
	MOVQ AX, 192(DI) // r1: integer return in RAX
	MOVQ DX, 200(DI) // r2: second integer return in RDX (9-16 byte structs)
	MOVQ X0, 128(DI) // f1: float return in XMM0

	// Restore stack and return
	XORL AX, AX          // no error (ignored by runtime.cgocall)
	ADDQ $STACK_SIZE, SP
	MOVQ BP, SP
	POPQ BP
	RET
