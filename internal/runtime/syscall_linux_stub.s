//go:build linux && amd64

#include "textflag.h"

// syscallStub is called by runtime.asmcgocall on g0 stack
// It receives pointer to callArgs struct in DI register
//
// func syscallStub(args unsafe.Pointer) uint64
TEXT ·syscallStub(SB), NOSPLIT|NOFRAME, $0
	// DI contains pointer to callArgs struct
	// Load function pointer
	MOVQ 0(DI), R11   // fn

	// Save args pointer
	MOVQ DI, R12

	// Load arguments into System V AMD64 ABI registers
	MOVQ 8(R12), DI   // a1 → RDI
	MOVQ 16(R12), SI  // a2 → RSI
	MOVQ 24(R12), DX  // a3 → RDX
	MOVQ 32(R12), CX  // a4 → RCX
	MOVQ 40(R12), R8  // a5 → R8
	MOVQ 48(R12), R9  // a6 → R9

	// Call C function
	// Stack is already aligned by asmcgocall!
	CALL R11

	// Save return value back to args.r1
	MOVQ AX, 56(R12)  // r1 = return value

	RET
