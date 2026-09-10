// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && goffi_universal && arm64

#include "textflag.h"

// func rawsyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1 uintptr)
//
// arm64 Linux syscall: number in R8, arguments in R0-R5, trap via SVC, result
// in R0. See the amd64 counterpart for why this exists.
TEXT ·rawsyscall6(SB), NOSPLIT|NOFRAME, $0-64
	MOVD trap+0(FP), R8
	MOVD a1+8(FP), R0
	MOVD a2+16(FP), R1
	MOVD a3+24(FP), R2
	MOVD a4+32(FP), R3
	MOVD a5+40(FP), R4
	MOVD a6+48(FP), R5
	SVC
	MOVD R0, r1+56(FP)
	RET
