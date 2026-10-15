// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && !goffi_static

#include "textflag.h"

// The _cgo_goffi_set*id_trampoline functions are what syscall.cgo_libc_set*id
// point at (see linux.go). runtime.syscall_cgocaller reaches them through
// runtime.asmcgocall on the g0 stack, passing a *argset in R0, which is also
// the first argument register of the Go function (ABIInternal) they call.

TEXT _cgo_goffi_setegid_trampoline(SB), NOSPLIT, $0-0
	MOVD ·x_cgo_goffi_setegid_call(SB), R9
	MOVD (R9), R9
	CALL R9
	RET

TEXT _cgo_goffi_seteuid_trampoline(SB), NOSPLIT, $0-0
	MOVD ·x_cgo_goffi_seteuid_call(SB), R9
	MOVD (R9), R9
	CALL R9
	RET

TEXT _cgo_goffi_setgid_trampoline(SB), NOSPLIT, $0-0
	MOVD ·x_cgo_goffi_setgid_call(SB), R9
	MOVD (R9), R9
	CALL R9
	RET

TEXT _cgo_goffi_setregid_trampoline(SB), NOSPLIT, $0-0
	MOVD ·x_cgo_goffi_setregid_call(SB), R9
	MOVD (R9), R9
	CALL R9
	RET

TEXT _cgo_goffi_setresgid_trampoline(SB), NOSPLIT, $0-0
	MOVD ·x_cgo_goffi_setresgid_call(SB), R9
	MOVD (R9), R9
	CALL R9
	RET

TEXT _cgo_goffi_setresuid_trampoline(SB), NOSPLIT, $0-0
	MOVD ·x_cgo_goffi_setresuid_call(SB), R9
	MOVD (R9), R9
	CALL R9
	RET

TEXT _cgo_goffi_setreuid_trampoline(SB), NOSPLIT, $0-0
	MOVD ·x_cgo_goffi_setreuid_call(SB), R9
	MOVD (R9), R9
	CALL R9
	RET

TEXT _cgo_goffi_setuid_trampoline(SB), NOSPLIT, $0-0
	MOVD ·x_cgo_goffi_setuid_call(SB), R9
	MOVD (R9), R9
	CALL R9
	RET

TEXT _cgo_goffi_setgroups_trampoline(SB), NOSPLIT, $0-0
	MOVD ·x_cgo_goffi_setgroups_call(SB), R9
	MOVD (R9), R9
	CALL R9
	RET
