// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && goffi_universal && arm64

#include "textflag.h"

// func setupUniversalTLS()
//
// arm64 counterpart of the amd64 shim. The thread pointer is TPIDR_EL0, which
// EL0 may read and write directly (no syscall). Go stores g at TPIDR_EL0+16
// (runtime.tls_g). If TPIDR_EL0 is unset (first, kernel-direct launch), point
// it at a scratch page so the compiler's g-reloads read mapped memory. No-op
// when it is already set (the re-executed launch).
//
// NOTE: arm64 is cross-compile-verified only in the development environment;
// it has not been run-tested. The logic mirrors the amd64 path.
TEXT ·setupUniversalTLS(SB), NOSPLIT, $0-0
	MRS	TPIDR_EL0, R0
	CBNZ	R0, tlsdone        // TLS already set up

	// p = mmap(0, 4096, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANON, -1, 0)
	MOVD	$0, R0
	MOVD	$4096, R1
	MOVD	$0x3, R2           // PROT_READ|PROT_WRITE
	MOVD	$0x22, R3          // MAP_PRIVATE|MAP_ANONYMOUS
	MOVD	$-1, R4
	MOVD	$0, R5
	MOVD	$222, R8           // SYS_mmap
	SVC
	TBNZ	$63, R0, tlsdone   // negative => -errno => mmap failed

	ADD	$2048, R0          // keep TPIDR_EL0+16 inside the page
	MSR	R0, TPIDR_EL0

tlsdone:
	RET
