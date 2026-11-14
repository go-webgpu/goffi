// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build arm64 && (linux || darwin || windows || freebsd)

#include "textflag.h"

// probe is called as a C function. It records X0-X7, D0-D7 and the 8 words
// at SP (the stack arguments) in probeRegs, then returns probeRet in X0
// and D0.
TEXT probe<>(SB), NOSPLIT|NOFRAME, $0
	MOVD	$·probeRegs(SB), R9
	MOVD	R0, 0(R9)
	MOVD	R1, 8(R9)
	MOVD	R2, 16(R9)
	MOVD	R3, 24(R9)
	MOVD	R4, 32(R9)
	MOVD	R5, 40(R9)
	MOVD	R6, 48(R9)
	MOVD	R7, 56(R9)
	FMOVD	F0, 64(R9)
	FMOVD	F1, 72(R9)
	FMOVD	F2, 80(R9)
	FMOVD	F3, 88(R9)
	FMOVD	F4, 96(R9)
	FMOVD	F5, 104(R9)
	FMOVD	F6, 112(R9)
	FMOVD	F7, 120(R9)
	MOVD	0(RSP), R10
	MOVD	R10, 128(R9)
	MOVD	8(RSP), R10
	MOVD	R10, 136(R9)
	MOVD	16(RSP), R10
	MOVD	R10, 144(R9)
	MOVD	24(RSP), R10
	MOVD	R10, 152(R9)
	MOVD	32(RSP), R10
	MOVD	R10, 160(R9)
	MOVD	40(RSP), R10
	MOVD	R10, 168(R9)
	MOVD	48(RSP), R10
	MOVD	R10, 176(R9)
	MOVD	56(RSP), R10
	MOVD	R10, 184(R9)
	MOVD	$·probeRet(SB), R10
	MOVD	(R10), R0
	FMOVD	(R10), F0
	RET

// errnoProbe stands in for __errno_location.
TEXT errnoProbe<>(SB), NOSPLIT|NOFRAME, $0
	MOVD	$·probeErrno(SB), R0
	RET

// func probeAddrs() (probe, errno uintptr)
TEXT ·probeAddrs(SB), NOSPLIT, $0-16
	MOVD	$probe<>(SB), R0
	MOVD	R0, probe+0(FP)
	MOVD	$errnoProbe<>(SB), R0
	MOVD	R0, errno+8(FP)
	RET
