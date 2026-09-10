// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && goffi_universal && amd64

#include "textflag.h"

// func rawsyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1 uintptr)
//
// A minimal, libc-free Linux syscall. Used only by the universal re-exec
// bridge, which runs inside x_cgo_init before any libc symbol is bound, so it
// must not route through the normal (libc-dependent) FFI path. Linux passes
// the 4th argument in R10 (not RCX); the return value comes back in AX.
TEXT ·rawsyscall6(SB), NOSPLIT|NOFRAME, $0-64
	MOVQ trap+0(FP), AX
	MOVQ a1+8(FP), DI
	MOVQ a2+16(FP), SI
	MOVQ a3+24(FP), DX
	MOVQ a4+32(FP), R10
	MOVQ a5+40(FP), R8
	MOVQ a6+48(FP), R9
	SYSCALL
	MOVQ AX, r1+56(FP)
	RET
