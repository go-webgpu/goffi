// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build amd64 && windows

package amd64

import "github.com/go-webgpu/goffi/types"

var winIntRegs = [4]int{rcx, rdx, r8, r9}

// bridgeCode returns the Win64 bridge for cif, or nil when an argument or
// the result is a struct. Like Execute, it zero-extends narrow integers and
// passes a float or double in the first four slots in both the integer and
// the XMM register, which is where a variadic callee reads it.
func bridgeCode(cif *types.CallInterface) []byte {
	n := len(cif.ArgTypes)
	if !bridgeReturn(cif.ReturnType) || n > maxWindowsSlots {
		return nil
	}
	for _, t := range cif.ArgTypes {
		if !bridgeKind(t) {
			return nil
		}
	}

	var a asm
	a.prologue(int32((32 + 8*max(0, n-4) + 15) &^ 15))
	a.saveFrame(rcx, false)
	for i := 4; i < n; i++ {
		a.loadArgPtr(i)
		a.loadInt(cif.ArgTypes[i].Kind, rax, r11, false)
		a.store64(rax, rsp, int32(32+8*(i-4)))
	}
	for i := range min(n, 4) {
		k := cif.ArgTypes[i].Kind
		a.loadArgPtr(i)
		a.loadInt(k, winIntRegs[i], r11, false)
		if k == types.FloatType || k == types.DoubleType {
			a.movq(i, winIntRegs[i])
		}
	}
	a.callFn()
	a.storeReturn(cif.ReturnType.Kind)
	a.emit(0x31, 0xc0) // xor eax, eax: no errno on Windows
	a.epilogue()
	return a.b
}

// movq copies general register r to XMM register x.
func (a *asm) movq(x, r int) {
	rex := byte(0x48)
	if x >= 8 {
		rex |= 4
	}
	if r >= 8 {
		rex |= 1
	}
	a.emit(0x66, rex, 0x0f, 0x6e, 0xc0|byte(x&7)<<3|byte(r&7))
}
