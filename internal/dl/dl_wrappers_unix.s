//go:build (linux || darwin) && amd64 && !cgo

#include "textflag.h"

// Assembly wrappers for dlopen/dlsym/dlerror using System V AMD64 ABI
// This calling convention is IDENTICAL on Linux and macOS, so we share
// the same implementation for both platforms.
//
// Reference: System V AMD64 ABI specification
// https://refspecs.linuxbase.org/elf/x86_64-abi-0.99.pdf

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
	PUSHQ BP
	MOVQ  SP, BP
	SUBQ  $16, SP
	MOVQ  DI, 0(SP)         // Save args pointer on stack

	// Load arguments for dlopen(path, mode)
	MOVQ 8(DI), DI          // path (offset 8) - LOAD FIRST!
	MOVQ 0(SP), R11         // Get args pointer
	MOVL 16(R11), SI        // mode (offset 16)
	MOVQ 0(R11), R10        // fn pointer (offset 0)

	// Call dlopen
	CALL R10

	// Store result
	MOVQ 0(SP), R11         // Restore args pointer
	MOVQ AX, 32(R11)        // result (offset 32)

	XORL AX, AX
	ADDQ $16, SP
	MOVQ BP, SP
	POPQ BP
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
	PUSHQ BP
	MOVQ  SP, BP
	SUBQ  $16, SP
	MOVQ  DI, 0(SP)         // Save args pointer on stack

	// Load arguments for dlsym(handle, symbol)
	MOVQ 8(DI), DI          // handle (offset 8) - LOAD FIRST!
	MOVQ 0(SP), R11         // Get args pointer
	MOVQ 16(R11), SI        // symbol (offset 16)
	MOVQ 0(R11), R10        // fn pointer (offset 0)

	// Call dlsym
	CALL R10

	// Store result
	MOVQ 0(SP), R11         // Restore args pointer
	MOVQ AX, 32(R11)        // result (offset 32)

	XORL AX, AX
	ADDQ $16, SP
	MOVQ BP, SP
	POPQ BP
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
	PUSHQ BP
	MOVQ  SP, BP
	SUBQ  $16, SP
	MOVQ  DI, 0(SP)         // Save args pointer on stack

	// Call dlerror (no arguments)
	MOVQ 0(DI), R10         // fn pointer (offset 0)
	CALL R10

	// Store result
	MOVQ 0(SP), R11         // Restore args pointer
	MOVQ AX, 8(R11)         // result (offset 8)

	XORL AX, AX
	ADDQ $16, SP
	MOVQ BP, SP
	POPQ BP
	RET
