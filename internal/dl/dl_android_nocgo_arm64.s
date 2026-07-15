// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build android && !cgo && arm64

#include "textflag.h"

TEXT android_dlopen_stub(SB), NOSPLIT|NOFRAME, $0-0
	B goffi_android_dlopen(SB)

TEXT android_dlsym_stub(SB), NOSPLIT|NOFRAME, $0-0
	B goffi_android_dlsym(SB)

TEXT android_dlerror_stub(SB), NOSPLIT|NOFRAME, $0-0
	B goffi_android_dlerror(SB)

TEXT android_strdup_stub(SB), NOSPLIT|NOFRAME, $0-0
	B goffi_android_strdup(SB)

TEXT android_free_stub(SB), NOSPLIT|NOFRAME, $0-0
	B goffi_android_free(SB)

// androidDlopenArgs offsets: fn=0, errorFn=8, strdupFn=16, path=24,
// mode=32, result=40, error=48.
GLOBL ·androidDlopenWrapperABI0(SB), NOPTR|RODATA, $8
DATA ·androidDlopenWrapperABI0(SB)/8, $androidDlopenWrapper(SB)

TEXT androidDlopenWrapper(SB), NOSPLIT|NOFRAME, $0
	SUB  $32, RSP, RSP
	MOVD R29, 0(RSP)
	MOVD R30, 8(RSP)
	MOVD R0, 16(RSP)
	MOVD RSP, R29

	MOVD R0, R9
	MOVD 24(R9), R0
	MOVD 32(R9), R1
	MOVD 0(R9), R10
	BL   (R10)
	MOVD 16(RSP), R9
	MOVD R0, 40(R9)
	CBNZ R0, android_dlopen_done

	MOVD 8(R9), R10
	BL   (R10)
	CBZ  R0, android_dlopen_done
	MOVD 16(RSP), R9
	MOVD 16(R9), R10
	BL   (R10)
	MOVD 16(RSP), R9
	MOVD R0, 48(R9)

android_dlopen_done:
	MOVD 8(RSP), R30
	MOVD 0(RSP), R29
	ADD  $32, RSP, RSP
	MOVD $0, R0
	RET

// androidDlsymArgs offsets: fn=0, errorFn=8, strdupFn=16, handle=24,
// name=32, result=40, error=48.
GLOBL ·androidDlsymWrapperABI0(SB), NOPTR|RODATA, $8
DATA ·androidDlsymWrapperABI0(SB)/8, $androidDlsymWrapper(SB)

TEXT androidDlsymWrapper(SB), NOSPLIT|NOFRAME, $0
	SUB  $32, RSP, RSP
	MOVD R29, 0(RSP)
	MOVD R30, 8(RSP)
	MOVD R0, 16(RSP)
	MOVD RSP, R29

	// Clear the caller thread's prior loader error before dlsym.
	MOVD R0, R9
	MOVD 8(R9), R10
	BL   (R10)
	MOVD 16(RSP), R9

	MOVD 24(R9), R0
	MOVD 32(R9), R1
	MOVD 0(R9), R10
	BL   (R10)
	MOVD 16(RSP), R9
	MOVD R0, 40(R9)
	CBNZ R0, android_dlsym_done

	MOVD 8(R9), R10
	BL   (R10)
	CBZ  R0, android_dlsym_done
	MOVD 16(RSP), R9
	MOVD 16(R9), R10
	BL   (R10)
	MOVD 16(RSP), R9
	MOVD R0, 48(R9)

android_dlsym_done:
	MOVD 8(RSP), R30
	MOVD 0(RSP), R29
	ADD  $32, RSP, RSP
	MOVD $0, R0
	RET

// androidFreeArgs offsets: fn=0, ptr=8.
GLOBL ·androidFreeWrapperABI0(SB), NOPTR|RODATA, $8
DATA ·androidFreeWrapperABI0(SB)/8, $androidFreeWrapper(SB)

TEXT androidFreeWrapper(SB), NOSPLIT|NOFRAME, $0
	SUB  $32, RSP, RSP
	MOVD R29, 0(RSP)
	MOVD R30, 8(RSP)
	MOVD R0, 16(RSP)
	MOVD RSP, R29

	MOVD R0, R9
	MOVD 8(R9), R0
	MOVD 0(R9), R10
	BL   (R10)

	MOVD 8(RSP), R30
	MOVD 0(RSP), R29
	ADD  $32, RSP, RSP
	MOVD $0, R0
	RET
