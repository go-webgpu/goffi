//go:build (linux || darwin) && arm64

#include "textflag.h"
#include "abi_arm64.h"

// syscall8 calls a C function with up to 8 integer and 8 float arguments.
// AAPCS64 calling convention (identical on Linux and macOS ARM64).
//
// syscall8 takes a pointer to syscall8Args struct:
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
//	f1    uintptr  // offset 72  (D0 input)
//	f2    uintptr  // offset 80  (D1 input)
//	f3    uintptr  // offset 88  (D2 input)
//	f4    uintptr  // offset 96  (D3 input)
//	f5    uintptr  // offset 104 (D4 input)
//	f6    uintptr  // offset 112 (D5 input)
//	f7    uintptr  // offset 120 (D6 input)
//	f8    uintptr  // offset 128 (D7 input)
//	r1    uintptr  // offset 136 (return X0)
//	r2    uintptr  // offset 144 (return X1)
//	fr1   uintptr  // offset 152 (return D0 for HFA)
//	fr2   uintptr  // offset 160 (return D1 for HFA)
//	fr3   uintptr  // offset 168 (return D2 for HFA)
//	fr4   uintptr  // offset 176 (return D3 for HFA)
//	r8    uintptr  // offset 184 (X8 - large struct return pointer)
// }
//
// syscall8 must be called on the g0 stack with runtime.cgocall.
GLOBL ·syscall8ABI0(SB), NOPTR|RODATA, $8
DATA ·syscall8ABI0(SB)/8, $syscall8(SB)

TEXT syscall8(SB), NOSPLIT|NOFRAME, $0
	// Save frame pointer and link register
	// R0 contains pointer to args struct (first argument in AAPCS64)
	SUB  $STACK_SIZE, RSP, RSP
	MOVD R29, (RSP)              // Save FP
	MOVD R30, 8(RSP)             // Save LR
	MOVD RSP, R29                // Set new FP
	MOVD R0, PTR_ADDRESS(RSP)    // Save args pointer

	// R9 = args pointer (use caller-saved temporary)
	MOVD R0, R9

	// Load float arguments into D0-D7 (offsets 72-128)
	FMOVD 72(R9), F0   // f1 -> D0
	FMOVD 80(R9), F1   // f2 -> D1
	FMOVD 88(R9), F2   // f3 -> D2
	FMOVD 96(R9), F3   // f4 -> D3
	FMOVD 104(R9), F4  // f5 -> D4
	FMOVD 112(R9), F5  // f6 -> D5
	FMOVD 120(R9), F6  // f7 -> D6
	FMOVD 128(R9), F7  // f8 -> D7

	// Load X8 for large struct return pointer (AAPCS64: X8 holds return address for >16 byte structs)
	MOVD 184(R9), R8  // r8 -> X8 (large struct return pointer)

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
	MOVD PTR_ADDRESS(RSP), R9

	// Save return values
	MOVD  R0, 136(R9)  // r1: integer return in X0
	MOVD  R1, 144(R9)  // r2: second integer return in X1
	FMOVD F0, 152(R9)  // fr1: D0 return for HFA
	FMOVD F1, 160(R9)  // fr2: D1 return for HFA
	FMOVD F2, 168(R9)  // fr3: D2 return for HFA
	FMOVD F3, 176(R9)  // fr4: D3 return for HFA

	// Restore frame and return
	MOVD 8(RSP), R30             // Restore LR
	MOVD (RSP), R29              // Restore FP
	ADD  $STACK_SIZE, RSP, RSP   // Restore SP
	MOVD $0, R0                  // no error (ignored by runtime.cgocall)
	RET
