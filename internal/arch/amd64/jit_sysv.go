// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build amd64 && (linux || darwin || freebsd)

package amd64

import "github.com/go-webgpu/goffi/types"

var sysvIntRegs = [sysvGPRs]int{rdi, rsi, rdx, rcx, r8, r9}

// bridgeCode returns the System V bridge for cif, or nil when an argument or
// the result is a struct. Arguments go where placeScalar puts them for
// Execute, extended as scalarWord extends them.
func bridgeCode(cif *types.CallInterface) []byte {
	if !bridgeReturn(cif.ReturnType) {
		return nil
	}
	type move struct {
		reg  int
		sse  bool
		word int
	}
	moves := make([]move, len(cif.ArgTypes))
	var alloc SysVAllocator
	for i, t := range cif.ArgTypes {
		if !bridgeKind(t) {
			return nil
		}
		reg, sse, word := alloc.placeScalar(t)
		moves[i] = move{reg, sse, word}
	}
	stackBytes := alloc.StackBytes()
	if stackBytes > MaxStackBytes {
		return nil
	}

	var a asm
	a.prologue(int32((stackBytes + 15) &^ 15))
	a.saveFrame(rdi, true)
	for i, m := range moves {
		if m.reg < 0 {
			a.loadArgPtr(i)
			a.loadInt(cif.ArgTypes[i].Kind, rax, r11, true)
			a.store64(rax, rsp, int32(8*m.word))
		}
	}
	for i, m := range moves {
		switch {
		case m.reg < 0:
		case m.sse:
			a.loadArgPtr(i)
			a.loadXMM(cif.ArgTypes[i].Kind, m.reg, r11)
		default:
			a.loadArgPtr(i)
			a.loadInt(cif.ArgTypes[i].Kind, sysvIntRegs[m.reg], r11, true)
		}
	}
	// AL bounds the vector registers a variadic callee saves.
	a.emit(0xb8) // mov eax, imm32
	a.imm32(int32(alloc.sse))
	a.callFn()
	a.storeReturn(cif.ReturnType.Kind)

	a.emit(0x31, 0xc0) // xor eax, eax
	a.load64(r11, rbp, localErrnoFn)
	a.emit(0x4d, 0x85, 0xdb) // test r11, r11
	a.emit(0x74, 0x05)       // jz past the next two instructions
	a.emit(0x41, 0xff, 0xd3) // call r11
	a.emit(0x8b, 0x00)       // mov eax, [rax]
	a.epilogue()
	return a.b
}

// loadXMM loads the float or double at [base] into XMM register x.
func (a *asm) loadXMM(k types.TypeKind, x, base int) {
	prefix := byte(0xf2) // movsd
	if k == types.FloatType {
		prefix = 0xf3 // movss
	}
	a.mem(prefix, false, []byte{0x0f, 0x10}, x, base, 0)
}
