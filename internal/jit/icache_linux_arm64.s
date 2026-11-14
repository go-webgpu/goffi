// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !android

#include "textflag.h"

// func flushICacheRange(start, end uintptr)
//
// The sequence the Arm Architecture Reference Manual gives for
// self-modifying code (B2.2.5): clean each data cache line to the point of
// unification, then invalidate each instruction cache line, with barriers
// between. CTR_EL0 gives the smallest line size of each cache.
TEXT ·flushICacheRange(SB), NOSPLIT|NOFRAME, $0-16
	MOVD	start+0(FP), R0
	MOVD	end+8(FP), R1
	MRS	CTR_EL0, R2
	MOVD	$4, R4

	UBFX	$16, R2, $4, R3 // DminLine: log2 of words per data cache line
	LSL	R3, R4, R3
	SUB	$1, R3, R5
	BIC	R5, R0, R6
dloop:
	WORD	$0xd50b7b26 // DC CVAU, R6
	ADD	R3, R6, R6
	CMP	R1, R6
	BLO	dloop
	DSB	$11 // ISH

	AND	$15, R2, R3 // IminLine
	LSL	R3, R4, R3
	SUB	$1, R3, R5
	BIC	R5, R0, R6
iloop:
	WORD	$0xd50b7526 // IC IVAU, R6
	ADD	R3, R6, R6
	CMP	R1, R6
	BLO	iloop
	DSB	$11
	ISB	$15
	RET
