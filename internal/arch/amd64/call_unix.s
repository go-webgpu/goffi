//go:build amd64 && (linux || darwin)

#include "textflag.h"

// func callUnix64(gpr []uint64, sse []float64, fn, stack unsafe.Pointer) uint64
TEXT ·callUnix64(SB), NOSPLIT, $0-48
    // Load arguments
    MOVQ gpr+0(FP), AX
    MOVQ sse+16(FP), BX
    MOVQ fn+32(FP), CX
    MOVQ stack+40(FP), DX

    // Load GPRs
    MOVQ 0(AX), DI
    MOVQ 8(AX), SI
    MOVQ 16(AX), R10
    MOVQ 24(AX), R8
    MOVQ 32(AX), R9
    MOVQ 40(AX), R11

    // Setup RDX and RCX
    MOVQ R10, DX
    MOVQ CX, R10

    // Load SSE registers
    MOVSD 0(BX), X0
    MOVSD 8(BX), X1
    MOVSD 16(BX), X2
    MOVSD 24(BX), X3
    MOVSD 32(BX), X4
    MOVSD 40(BX), X5
    MOVSD 48(BX), X6
    MOVSD 56(BX), X7

    // Setup stack
    MOVQ DX, SP
    SUBQ $8, SP

    // Call function
    CALL R10

    // Return value
    MOVQ AX, ret+48(FP)
    RET
    