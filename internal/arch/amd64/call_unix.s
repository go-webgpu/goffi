//go:build amd64 && (linux || darwin)

#include "textflag.h"

// func callUnix64(gpr []uint64, sse []float64, fn uintptr) uint64
// System V AMD64 ABI calling convention (identical on Linux and macOS)
// EXPERIMENT: Try direct call WITHOUT stack manipulation
TEXT ·callUnix64(SB), NOSPLIT, $0-56
	// Load arguments
	MOVQ gpr+0(FP), AX   // GPR array pointer
	MOVQ sse+24(FP), BX  // SSE array pointer
	MOVQ fn+48(FP), R11  // Function pointer

	// Load GPRs
	MOVQ 0(AX), DI
	MOVQ 8(AX), SI
	MOVQ 16(AX), DX
	MOVQ 24(AX), CX
	MOVQ 32(AX), R8
	MOVQ 40(AX), R9

	// Load SSE registers
	MOVSD 0(BX), X0
	MOVSD 8(BX), X1
	MOVSD 16(BX), X2
	MOVSD 24(BX), X3
	MOVSD 32(BX), X4
	MOVSD 40(BX), X5
	MOVSD 48(BX), X6
	MOVSD 56(BX), X7

	// Direct call - let's see what happens!
	CALL R11

	// Return value
	MOVQ AX, ret+56(FP)
	RET
