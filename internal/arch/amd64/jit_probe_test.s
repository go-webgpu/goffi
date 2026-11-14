// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build amd64 && (linux || darwin || freebsd || windows)

#include "textflag.h"

// probe is called as a C function. It records RDI, RSI, RDX, RCX, R8, R9,
// RAX, XMM0-XMM7 and the 16 words above its return address in probeRegs,
// then returns probeRet in RAX and XMM0.
TEXT probe<>(SB), NOSPLIT|NOFRAME, $0
	MOVQ	DI, ·probeRegs+0(SB)
	MOVQ	SI, ·probeRegs+8(SB)
	MOVQ	DX, ·probeRegs+16(SB)
	MOVQ	CX, ·probeRegs+24(SB)
	MOVQ	R8, ·probeRegs+32(SB)
	MOVQ	R9, ·probeRegs+40(SB)
	MOVQ	AX, ·probeRegs+48(SB)
	MOVQ	X0, ·probeRegs+56(SB)
	MOVQ	X1, ·probeRegs+64(SB)
	MOVQ	X2, ·probeRegs+72(SB)
	MOVQ	X3, ·probeRegs+80(SB)
	MOVQ	X4, ·probeRegs+88(SB)
	MOVQ	X5, ·probeRegs+96(SB)
	MOVQ	X6, ·probeRegs+104(SB)
	MOVQ	X7, ·probeRegs+112(SB)
	MOVQ	8(SP), R10
	MOVQ	R10, ·probeRegs+120(SB)
	MOVQ	16(SP), R10
	MOVQ	R10, ·probeRegs+128(SB)
	MOVQ	24(SP), R10
	MOVQ	R10, ·probeRegs+136(SB)
	MOVQ	32(SP), R10
	MOVQ	R10, ·probeRegs+144(SB)
	MOVQ	40(SP), R10
	MOVQ	R10, ·probeRegs+152(SB)
	MOVQ	48(SP), R10
	MOVQ	R10, ·probeRegs+160(SB)
	MOVQ	56(SP), R10
	MOVQ	R10, ·probeRegs+168(SB)
	MOVQ	64(SP), R10
	MOVQ	R10, ·probeRegs+176(SB)
	MOVQ	72(SP), R10
	MOVQ	R10, ·probeRegs+184(SB)
	MOVQ	80(SP), R10
	MOVQ	R10, ·probeRegs+192(SB)
	MOVQ	88(SP), R10
	MOVQ	R10, ·probeRegs+200(SB)
	MOVQ	96(SP), R10
	MOVQ	R10, ·probeRegs+208(SB)
	MOVQ	104(SP), R10
	MOVQ	R10, ·probeRegs+216(SB)
	MOVQ	112(SP), R10
	MOVQ	R10, ·probeRegs+224(SB)
	MOVQ	120(SP), R10
	MOVQ	R10, ·probeRegs+232(SB)
	MOVQ	128(SP), R10
	MOVQ	R10, ·probeRegs+240(SB)
	MOVQ	·probeRet(SB), AX
	MOVQ	·probeRet(SB), X0
	RET

// errnoProbe stands in for __errno_location.
TEXT errnoProbe<>(SB), NOSPLIT|NOFRAME, $0
	LEAQ	·probeErrno(SB), AX
	RET

// func probeAddrs() (probe, errno uintptr)
TEXT ·probeAddrs(SB), NOSPLIT, $0-16
	LEAQ	probe<>(SB), AX
	MOVQ	AX, probe+0(FP)
	LEAQ	errnoProbe<>(SB), AX
	MOVQ	AX, errno+8(FP)
	RET
