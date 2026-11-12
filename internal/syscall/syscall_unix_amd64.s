//go:build (linux || darwin || freebsd) && amd64

#include "textflag.h"
#include "abi_amd64.h"

// syscallN calls a C function with 6 integer register arguments, 8 SSE
// register arguments and a variable number of 8-byte stack words.
// System V AMD64 ABI calling convention (identical on Linux, macOS, FreeBSD).
//
// syscallN takes a pointer to syscallArgs struct:
// struct {
//	fn      uintptr  // offset 0
//	a1      uintptr  // offset 8   (RDI)
//	a2      uintptr  // offset 16  (RSI)
//	a3      uintptr  // offset 24  (RDX)
//	a4      uintptr  // offset 32  (RCX)
//	a5      uintptr  // offset 40  (R8)
//	a6      uintptr  // offset 48  (R9)
//	stack   uintptr  // offset 56  (address of the first stack word)
//	nstack  uintptr  // offset 64  (number of stack words)
//	f1      uintptr  // offset 72  (XMM0 bit pattern)
//	f2      uintptr  // offset 80  (XMM1)
//	f3      uintptr  // offset 88  (XMM2)
//	f4      uintptr  // offset 96  (XMM3)
//	f5      uintptr  // offset 104 (XMM4)
//	f6      uintptr  // offset 112 (XMM5)
//	f7      uintptr  // offset 120 (XMM6)
//	f8      uintptr  // offset 128 (XMM7)
//	r1      uintptr  // offset 136 (RAX return)
//	r2      uintptr  // offset 144 (RDX return, 9-16 byte struct)
//	errno   uintptr  // offset 152 (captured C errno; 0 if errnoFn == 0)
//	errnoFn uintptr  // offset 160 (address of __errno_location/__error; 0 = skip)
// }
//
// syscallN must be called on the g0 stack with runtime.cgocall.
//
// Stack frame: BP is pushed, then nstack words rounded up to 16 bytes are
// reserved below it and the stack words are copied there, stack[0] at SP.
// asmcgocall aligns SP to 16 bytes before its CALL, so after PUSHQ BP the SP
// is 16-byte aligned again and stays so at our CALL, as the ABI requires. The
// args pointer is kept in R12, which is callee-saved, across the C call; SP
// is restored from BP. The caller bounds nstack (amd64.MaxStackBytes).
GLOBL ·syscallNABI0(SB), NOPTR|RODATA, $8
DATA ·syscallNABI0(SB)/8, $syscallN(SB)

TEXT syscallN(SB), NOSPLIT|NOFRAME, $0
	PUSHQ BP
	MOVQ  SP, BP
	MOVQ  DI, R12 // R12 = args pointer (callee-saved, survives the C call)

	// Reserve round16(nstack*8) bytes and copy the stack words, stack[0] at SP.
	MOVQ  64(R12), CX // CX = nstack
	MOVQ  CX, AX
	SHLQ  $3, AX
	ADDQ  $15, AX
	ANDQ  $~15, AX
	SUBQ  AX, SP
	MOVQ  56(R12), SI // SI = &stack[0]
	XORQ  DX, DX

copy_loop:
	CMPQ DX, CX
	JGE  copy_done
	MOVQ (SI)(DX*8), AX
	MOVQ AX, (SP)(DX*8)
	INCQ DX
	JMP  copy_loop

copy_done:
	// Load float arguments into XMM0-XMM7 (offsets 72-128)
	MOVQ 72(R12), X0  // f1
	MOVQ 80(R12), X1  // f2
	MOVQ 88(R12), X2  // f3
	MOVQ 96(R12), X3  // f4
	MOVQ 104(R12), X4 // f5
	MOVQ 112(R12), X5 // f6
	MOVQ 120(R12), X6 // f7
	MOVQ 128(R12), X7 // f8

	// Load integer arguments into GP registers (System V AMD64 ABI, offsets 8-48)
	MOVQ 8(R12), DI  // a1 -> RDI
	MOVQ 16(R12), SI // a2 -> RSI
	MOVQ 24(R12), DX // a3 -> RDX
	MOVQ 32(R12), CX // a4 -> RCX
	MOVQ 40(R12), R8 // a5 -> R8
	MOVQ 48(R12), R9 // a6 -> R9

	// AL is an upper bound on the vector registers a variadic callee reads
	// arguments from. 0 tells it XMM0-XMM7 hold none, so it reads garbage for
	// a variadic double. 8 is always correct; a callee that is not variadic
	// ignores AL.
	MOVL $8, AX

	// Load function pointer and call (offset 0)
	MOVQ 0(R12), R10
	CALL R10

	// Save C function return values; R12 still holds the args pointer.
	MOVQ AX, 136(R12) // r1: integer return in RAX
	MOVQ DX, 144(R12) // r2: second integer return in RDX (9-16 byte structs)
	MOVQ X0, 72(R12)  // f1: float return in XMM0
	MOVQ X1, 80(R12)  // f2: XMM1 — second SSE return for 9-16B all-float struct returns

	// Errno capture (conditional): only when errnoFn (offset 160) is non-zero.
	// Safe window: we are still on g0, same OS thread as the C call.
	MOVQ  160(R12), R13 // R13 = errnoFn address
	TESTQ R13, R13
	JZ    errno_done
	CALL  R13           // __errno_location()/__error() → RAX = &errno (int*)
	MOVL  (AX), AX      // AX = *(&errno) as int32, zero-extended to 64 bits
	MOVQ  AX, 152(R12)  // args->errno = captured errno value

errno_done:
	// Restore stack and return
	XORL AX, AX // no error (ignored by runtime.cgocall)
	MOVQ BP, SP
	POPQ BP
	RET
