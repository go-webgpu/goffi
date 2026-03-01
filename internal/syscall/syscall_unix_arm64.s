//go:build (linux || darwin) && arm64

#include "textflag.h"
#include "abi_arm64.h"

// syscallN calls a C function with up to 8 integer, 8 float, and 7 stack arguments.
// AAPCS64 calling convention (identical on Linux and macOS ARM64).
// Follows purego's syscall15X pattern for ARM64.
//
// syscallN takes a pointer to syscallArgs struct (offsets verified by check_arm64.go):
// struct {
//	fn    uintptr  // offset 0
//	a1    uintptr  // offset 8   (X0)
//	a2    uintptr  // offset 16  (X1)
//	a3    uintptr  // offset 24  (X2)
//	a4    uintptr  // offset 32  (X3)
//	a5    uintptr  // offset 40  (X4)
//	a6    uintptr  // offset 48  (X5)
//	a7    uintptr  // offset 56  (X6)
//	a8    uintptr  // offset 64  (X7)
//	a9    uintptr  // offset 72  (stack[0])
//	a10   uintptr  // offset 80  (stack[1])
//	a11   uintptr  // offset 88  (stack[2])
//	a12   uintptr  // offset 96  (stack[3])
//	a13   uintptr  // offset 104 (stack[4])
//	a14   uintptr  // offset 112 (stack[5])
//	a15   uintptr  // offset 120 (stack[6])
//	f1    uintptr  // offset 128 (D0 input)
//	f2    uintptr  // offset 136 (D1 input)
//	f3    uintptr  // offset 144 (D2 input)
//	f4    uintptr  // offset 152 (D3 input)
//	f5    uintptr  // offset 160 (D4 input)
//	f6    uintptr  // offset 168 (D5 input)
//	f7    uintptr  // offset 176 (D6 input)
//	f8    uintptr  // offset 184 (D7 input)
//	r1    uintptr  // offset 192 (return X0)
//	r2    uintptr  // offset 200 (return X1)
//	fr1   uintptr  // offset 208 (return D0 for HFA or float)
//	fr2   uintptr  // offset 216 (return D1 for HFA)
//	fr3   uintptr  // offset 224 (return D2 for HFA)
//	fr4   uintptr  // offset 232 (return D3 for HFA)
//	r8    uintptr  // offset 240 (X8 - large struct return pointer)
// }
//
// Stack frame layout (total STACK_SIZE = 96 bytes, 16-byte aligned):
//   RSP+0  .. RSP+55 : 7 stack-spill slots for a9-a15 (7 * 8 = 56 bytes)
//   RSP+56 .. RSP+63 : padding (8 bytes for 16-byte alignment)
//   RSP+64 .. RSP+71 : saved FP (R29)
//   RSP+72 .. RSP+79 : saved LR (R30)
//   RSP+80 .. RSP+87 : saved args pointer (PTR_ADDRESS = 80)
//   RSP+88 .. RSP+95 : padding
//   Total: 96 bytes
//
// syscallN must be called on the g0 stack with runtime.cgocall.
GLOBL ·syscallNABI0(SB), NOPTR|RODATA, $8
DATA ·syscallNABI0(SB)/8, $syscallN(SB)

TEXT syscallN(SB), NOSPLIT|NOFRAME, $0
	// Save frame pointer and link register.
	// R0 = pointer to syscallArgs struct (first argument in AAPCS64).
	SUB  $STACK_SIZE, RSP, RSP
	MOVD R29, 64(RSP)           // Save FP at RSP+64
	MOVD R30, 72(RSP)           // Save LR at RSP+72
	MOVD RSP, R29               // Set new FP = current RSP
	MOVD R0, 80(RSP)            // Save args pointer at RSP+80 (PTR_ADDRESS)

	// R9 = args pointer (caller-saved temporary register in AAPCS64)
	MOVD R0, R9

	// Load float arguments into D0-D7 (offsets 128-184)
	FMOVD 128(R9), F0  // f1 -> D0
	FMOVD 136(R9), F1  // f2 -> D1
	FMOVD 144(R9), F2  // f3 -> D2
	FMOVD 152(R9), F3  // f4 -> D3
	FMOVD 160(R9), F4  // f5 -> D4
	FMOVD 168(R9), F5  // f6 -> D5
	FMOVD 176(R9), F6  // f7 -> D6
	FMOVD 184(R9), F7  // f8 -> D7

	// Load X8 for large struct return pointer (AAPCS64: X8 holds sret address)
	MOVD 240(R9), R8  // r8 -> X8

	// Push stack-spill arguments a9-a15 onto the stack (offsets 72-120).
	// AAPCS64: additional integer args go on the stack in order (a9 at SP+0, ...).
	MOVD 72(R9), R11
	MOVD R11, 0(RSP)   // push a9  -> stack[0]
	MOVD 80(R9), R11
	MOVD R11, 8(RSP)   // push a10 -> stack[1]
	MOVD 88(R9), R11
	MOVD R11, 16(RSP)  // push a11 -> stack[2]
	MOVD 96(R9), R11
	MOVD R11, 24(RSP)  // push a12 -> stack[3]
	MOVD 104(R9), R11
	MOVD R11, 32(RSP)  // push a13 -> stack[4]
	MOVD 112(R9), R11
	MOVD R11, 40(RSP)  // push a14 -> stack[5]
	MOVD 120(R9), R11
	MOVD R11, 48(RSP)  // push a15 -> stack[6]

	// Load integer arguments into X0-X7 (offsets 8-64)
	MOVD 8(R9), R0    // a1 -> X0
	MOVD 16(R9), R1   // a2 -> X1
	MOVD 24(R9), R2   // a3 -> X2
	MOVD 32(R9), R3   // a4 -> X3
	MOVD 40(R9), R4   // a5 -> X4
	MOVD 48(R9), R5   // a6 -> X5
	MOVD 56(R9), R6   // a7 -> X6
	MOVD 64(R9), R7   // a8 -> X7

	// Load function pointer into R10 (IP0) and call
	MOVD 0(R9), R10   // fn
	BL   (R10)

	// Get the args pointer back
	MOVD 80(RSP), R9  // PTR_ADDRESS = 80

	// Save return values (offsets verified by check_arm64.go)
	MOVD  R0, 192(R9)  // r1: integer return in X0
	MOVD  R1, 200(R9)  // r2: second integer return in X1
	FMOVD F0, 208(R9)  // fr1: D0 return for HFA or float
	FMOVD F1, 216(R9)  // fr2: D1 return for HFA
	FMOVD F2, 224(R9)  // fr3: D2 return for HFA
	FMOVD F3, 232(R9)  // fr4: D3 return for HFA

	// Restore frame and return
	MOVD 72(RSP), R30            // Restore LR
	MOVD 64(RSP), R29            // Restore FP
	ADD  $STACK_SIZE, RSP, RSP   // Restore SP
	MOVD $0, R0                  // no error (ignored by runtime.cgocall)
	RET
