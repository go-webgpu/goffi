// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && !goffi_static

#include "textflag.h"

// The _cgo_goffi_set*id_trampoline functions are what syscall.cgo_libc_set*id
// point at (see linux.go). runtime.syscall_cgocaller reaches them through
// runtime.asmcgocall on the g0 stack, passing a *argset in DI (C ABI). They
// forward it in AX to the matching Go function (ABIInternal).

TEXT _cgo_goffi_setegid_trampoline(SB), NOSPLIT, $8
	MOVQ DI, AX
	MOVQ ·x_cgo_goffi_setegid_call(SB), R11
	MOVQ (R11), R11
	CALL R11
	RET

TEXT _cgo_goffi_seteuid_trampoline(SB), NOSPLIT, $8
	MOVQ DI, AX
	MOVQ ·x_cgo_goffi_seteuid_call(SB), R11
	MOVQ (R11), R11
	CALL R11
	RET

TEXT _cgo_goffi_setgid_trampoline(SB), NOSPLIT, $8
	MOVQ DI, AX
	MOVQ ·x_cgo_goffi_setgid_call(SB), R11
	MOVQ (R11), R11
	CALL R11
	RET

TEXT _cgo_goffi_setregid_trampoline(SB), NOSPLIT, $8
	MOVQ DI, AX
	MOVQ ·x_cgo_goffi_setregid_call(SB), R11
	MOVQ (R11), R11
	CALL R11
	RET

TEXT _cgo_goffi_setresgid_trampoline(SB), NOSPLIT, $8
	MOVQ DI, AX
	MOVQ ·x_cgo_goffi_setresgid_call(SB), R11
	MOVQ (R11), R11
	CALL R11
	RET

TEXT _cgo_goffi_setresuid_trampoline(SB), NOSPLIT, $8
	MOVQ DI, AX
	MOVQ ·x_cgo_goffi_setresuid_call(SB), R11
	MOVQ (R11), R11
	CALL R11
	RET

TEXT _cgo_goffi_setreuid_trampoline(SB), NOSPLIT, $8
	MOVQ DI, AX
	MOVQ ·x_cgo_goffi_setreuid_call(SB), R11
	MOVQ (R11), R11
	CALL R11
	RET

TEXT _cgo_goffi_setuid_trampoline(SB), NOSPLIT, $8
	MOVQ DI, AX
	MOVQ ·x_cgo_goffi_setuid_call(SB), R11
	MOVQ (R11), R11
	CALL R11
	RET

TEXT _cgo_goffi_setgroups_trampoline(SB), NOSPLIT, $8
	MOVQ DI, AX
	MOVQ ·x_cgo_goffi_setgroups_call(SB), R11
	MOVQ (R11), R11
	CALL R11
	RET
