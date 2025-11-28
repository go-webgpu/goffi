//go:build (linux || darwin) && arm64

#include "textflag.h"

// Assembly wrappers for dlopen/dlsym/dlerror using AAPCS64 ABI
// This calling convention is IDENTICAL on Linux and macOS ARM64.
//
// Reference: ARM64 Procedure Call Standard (AAPCS64)
// https://developer.arm.com/documentation/ihi0055/latest

// dlopen_wrapper calls dlopen(path, mode)
//
// Args struct layout:
//   fn     uintptr  // offset 0
//   path   *byte    // offset 8
//   mode   int      // offset 16
//   _pad   int      // offset 24
//   result uintptr  // offset 32
//
GLOBL ·dlopen_wrapperABI0(SB), NOPTR|RODATA, $8
DATA ·dlopen_wrapperABI0(SB)/8, $dlopen_wrapper(SB)

TEXT dlopen_wrapper(SB), NOSPLIT|NOFRAME, $0
	// R0 contains args pointer (first argument in AAPCS64)
	// Save frame pointer, link register, and args pointer
	SUB  $32, RSP, RSP
	MOVD R29, (RSP)           // Save FP
	MOVD R30, 8(RSP)          // Save LR
	MOVD R0, 16(RSP)          // Save args pointer
	MOVD RSP, R29             // Set new FP

	// R9 = args pointer (callee-saved temp)
	MOVD R0, R9

	// Load arguments for dlopen(path, mode)
	MOVD 8(R9), R0            // path (offset 8) -> X0
	MOVD 16(R9), R1           // mode (offset 16) -> X1
	MOVD 0(R9), R10           // fn pointer (offset 0)

	// Call dlopen
	BL (R10)

	// Restore args pointer and store result
	MOVD 16(RSP), R9          // Restore args pointer
	MOVD R0, 32(R9)           // result (offset 32)

	// Restore and return
	MOVD 8(RSP), R30          // Restore LR
	MOVD (RSP), R29           // Restore FP
	ADD  $32, RSP, RSP
	MOVD $0, R0               // no error
	RET

// dlsym_wrapper calls dlsym(handle, symbol)
//
// Args struct layout:
//   fn     uintptr  // offset 0
//   handle uintptr  // offset 8
//   symbol *byte    // offset 16
//   _pad   int      // offset 24
//   result uintptr  // offset 32
//
GLOBL ·dlsym_wrapperABI0(SB), NOPTR|RODATA, $8
DATA ·dlsym_wrapperABI0(SB)/8, $dlsym_wrapper(SB)

TEXT dlsym_wrapper(SB), NOSPLIT|NOFRAME, $0
	// R0 contains args pointer
	SUB  $32, RSP, RSP
	MOVD R29, (RSP)           // Save FP
	MOVD R30, 8(RSP)          // Save LR
	MOVD R0, 16(RSP)          // Save args pointer
	MOVD RSP, R29             // Set new FP

	MOVD R0, R9

	// Load arguments for dlsym(handle, symbol)
	MOVD 8(R9), R0            // handle (offset 8) -> X0
	MOVD 16(R9), R1           // symbol (offset 16) -> X1
	MOVD 0(R9), R10           // fn pointer (offset 0)

	// Call dlsym
	BL (R10)

	// Restore args pointer and store result
	MOVD 16(RSP), R9
	MOVD R0, 32(R9)           // result (offset 32)

	// Restore and return
	MOVD 8(RSP), R30
	MOVD (RSP), R29
	ADD  $32, RSP, RSP
	MOVD $0, R0
	RET

// dlerror_wrapper calls dlerror()
//
// Args struct layout:
//   fn     uintptr  // offset 0
//   result *byte    // offset 8
//
GLOBL ·dlerror_wrapperABI0(SB), NOPTR|RODATA, $8
DATA ·dlerror_wrapperABI0(SB)/8, $dlerror_wrapper(SB)

TEXT dlerror_wrapper(SB), NOSPLIT|NOFRAME, $0
	// R0 contains args pointer
	SUB  $32, RSP, RSP
	MOVD R29, (RSP)           // Save FP
	MOVD R30, 8(RSP)          // Save LR
	MOVD R0, 16(RSP)          // Save args pointer
	MOVD RSP, R29             // Set new FP

	MOVD R0, R9

	// Call dlerror (no arguments)
	MOVD 0(R9), R10           // fn pointer (offset 0)
	BL (R10)

	// Restore args pointer and store result
	MOVD 16(RSP), R9
	MOVD R0, 8(R9)            // result (offset 8)

	// Restore and return
	MOVD 8(RSP), R30
	MOVD (RSP), R29
	ADD  $32, RSP, RSP
	MOVD $0, R0
	RET
