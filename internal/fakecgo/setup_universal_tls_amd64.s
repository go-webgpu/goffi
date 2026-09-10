// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && goffi_universal && amd64

#include "textflag.h"

// func setupUniversalTLS()
//
// If %fs is unset (first, kernel-direct launch), point it at a scratch page so
// the compiler's post-ABI0-call `MOVQ FS:-8, R14` g-reloads read mapped memory.
// No-op when %fs is already set (the re-executed launch). Never touches R14 (g)
// or BP; SYSCALL's clobber of RCX/R11 is irrelevant here.
TEXT ·setupUniversalTLS(SB), NOSPLIT, $0-0
	// current = arch_prctl(ARCH_GET_FS, &utlsScratch)
	MOVQ	$0, ·utlsScratch(SB)
	MOVQ	$158, AX           // SYS_arch_prctl
	MOVQ	$0x1003, DI        // ARCH_GET_FS
	LEAQ	·utlsScratch(SB), SI
	SYSCALL
	MOVQ	·utlsScratch(SB), AX
	TESTQ	AX, AX
	JNE	tlsdone            // TLS already set up

	// p = mmap(0, 4096, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANON, -1, 0)
	MOVQ	$9, AX             // SYS_mmap
	MOVQ	$0, DI
	MOVQ	$4096, SI
	MOVQ	$0x3, DX           // PROT_READ|PROT_WRITE
	MOVQ	$0x22, R10         // MAP_PRIVATE|MAP_ANONYMOUS
	MOVQ	$-1, R8
	MOVQ	$0, R9
	SYSCALL
	CMPQ	AX, $-4096
	JAE	tlsdone            // mmap failed; nothing better to do

	// arch_prctl(ARCH_SET_FS, p+2048) -- offset keeps FS:-8 inside the page
	ADDQ	$2048, AX
	MOVQ	AX, SI
	MOVQ	$158, AX           // SYS_arch_prctl
	MOVQ	$0x1002, DI        // ARCH_SET_FS
	SYSCALL

tlsdone:
	RET
