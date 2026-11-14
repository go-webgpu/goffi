// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build amd64 && (linux || darwin || freebsd || windows)

package amd64

import (
	"encoding/binary"
	"unsafe"

	"github.com/go-webgpu/goffi/internal/jit"
	"github.com/go-webgpu/goffi/types"
)

// Compile implements arch.Compiler: it returns a call bridge for cif (see
// package jit), or nil when cif needs the generic call path.
func (i *Implementation) Compile(cif *types.CallInterface) unsafe.Pointer {
	code := bridgeCode(cif)
	if code == nil {
		return nil
	}
	return unsafe.Pointer(jit.Install(code))
}

// The bridges of jit_sysv.go and jit_windows.go share one frame:
//
//	push rbp; mov rbp, rsp      the prologue package jit's Windows unwind
//	                            information describes
//	sub rsp, n; and rsp, -16
//	[rbp-8]   fn                copied from the jit frame, which the C
//	[rbp-16]  rvalue            call may invalidate (a callback can move
//	[rbp-24]  errnoFn           the goroutine stack it lives on)
//	[rsp]     outgoing stack arguments
const (
	localFn      = -8
	localRvalue  = -16
	localErrnoFn = -24
	localsSize   = 32
)

const (
	rax = 0
	rcx = 1
	rdx = 2
	rsp = 4
	rbp = 5
	rsi = 6
	rdi = 7
	r8  = 8
	r9  = 9
	r10 = 10
	r11 = 11
)

// bridgeKind reports whether a bridge passes and returns values of kind k.
func bridgeKind(t *types.TypeDescriptor) bool {
	if t == nil {
		return false
	}
	switch t.Kind {
	case types.IntType, types.FloatType, types.DoubleType,
		types.UInt8Type, types.SInt8Type, types.UInt16Type, types.SInt16Type,
		types.UInt32Type, types.SInt32Type, types.UInt64Type, types.SInt64Type,
		types.PointerType:
		return true
	}
	return false
}

func bridgeReturn(t *types.TypeDescriptor) bool {
	return t != nil && (t.Kind == types.VoidType || bridgeKind(t))
}

type asm struct{ b []byte }

func (a *asm) emit(bs ...byte) { a.b = append(a.b, bs...) }

func (a *asm) imm32(v int32) { a.b = binary.LittleEndian.AppendUint32(a.b, uint32(v)) }

// mem emits [prefix] [REX] opcode ModRM [SIB] [disp] with reg in ModRM.reg
// and the memory operand [base+disp].
func (a *asm) mem(prefix byte, w bool, op []byte, reg, base int, disp int32) {
	if prefix != 0 {
		a.emit(prefix)
	}
	rex := byte(0x40)
	if w {
		rex |= 8
	}
	if reg >= 8 {
		rex |= 4
	}
	if base >= 8 {
		rex |= 1
	}
	if rex != 0x40 {
		a.emit(rex)
	}
	a.emit(op...)
	modrm := byte(reg&7)<<3 | byte(base&7)
	short := disp >= -128 && disp <= 127
	switch {
	case disp == 0 && base&7 != rbp:
	case short:
		modrm |= 0x40
	default:
		modrm |= 0x80
	}
	a.emit(modrm)
	if base&7 == rsp {
		a.emit(0x24)
	}
	switch {
	case disp == 0 && base&7 != rbp:
	case short:
		a.emit(byte(int8(disp)))
	default:
		a.imm32(disp)
	}
}

func (a *asm) load64(dst, base int, disp int32)  { a.mem(0, true, []byte{0x8b}, dst, base, disp) }
func (a *asm) store64(src, base int, disp int32) { a.mem(0, true, []byte{0x89}, src, base, disp) }

// loadInt loads the value of kind k at [base] into the general register
// dst: a float as its 32 bits, an integer narrower than 64 bits sign- or
// zero-extended by its signedness, or zero-extended when signed is false.
func (a *asm) loadInt(k types.TypeKind, dst, base int, signed bool) {
	switch k {
	case types.SInt8Type, types.UInt8Type:
		if signed && k == types.SInt8Type {
			a.mem(0, true, []byte{0x0f, 0xbe}, dst, base, 0) // movsx r64, byte
		} else {
			a.mem(0, false, []byte{0x0f, 0xb6}, dst, base, 0) // movzx r32, byte
		}
	case types.SInt16Type, types.UInt16Type:
		if signed && k == types.SInt16Type {
			a.mem(0, true, []byte{0x0f, 0xbf}, dst, base, 0) // movsx r64, word
		} else {
			a.mem(0, false, []byte{0x0f, 0xb7}, dst, base, 0) // movzx r32, word
		}
	case types.SInt32Type, types.IntType, types.UInt32Type, types.FloatType:
		if signed && k != types.UInt32Type && k != types.FloatType {
			a.mem(0, true, []byte{0x63}, dst, base, 0) // movsxd r64, dword
		} else {
			a.mem(0, false, []byte{0x8b}, dst, base, 0) // mov r32, dword
		}
	default:
		a.load64(dst, base, 0)
	}
}

// prologue sets up the frame with out bytes (a multiple of 16) for
// outgoing arguments.
func (a *asm) prologue(out int32) {
	a.emit(0x55, 0x48, 0x89, 0xe5) // push rbp; mov rbp, rsp
	a.emit(0x48, 0x81, 0xec)       // sub rsp, imm32
	a.imm32(localsSize + out)
	a.emit(0x48, 0x83, 0xe4, 0xf0) // and rsp, -16
}

// saveFrame copies the jit frame at [fp] to the locals and loads &avalue[0]
// into R10.
func (a *asm) saveFrame(fp int, errno bool) {
	a.load64(rax, fp, 0)
	a.store64(rax, rbp, localFn)
	a.load64(rax, fp, 16)
	a.store64(rax, rbp, localRvalue)
	if errno {
		a.load64(rax, fp, 24)
		a.store64(rax, rbp, localErrnoFn)
	}
	a.load64(r10, fp, 8)
}

// loadArgPtr loads avalue[i] into R11.
func (a *asm) loadArgPtr(i int) { a.load64(r11, r10, int32(8*i)) }

func (a *asm) callFn() { a.mem(0, false, []byte{0xff}, 2, rbp, localFn) } // call [rbp-8]

// storeReturn stores the result of kind k from RAX or XMM0 to rvalue, at
// its own size, unless rvalue is nil.
func (a *asm) storeReturn(k types.TypeKind) {
	if k == types.VoidType {
		return
	}
	a.load64(r11, rbp, localRvalue)
	a.emit(0x4d, 0x85, 0xdb) // test r11, r11
	a.emit(0x74, 0)          // jz past the store
	from := len(a.b)
	switch k {
	case types.FloatType:
		a.mem(0xf3, false, []byte{0x0f, 0x11}, 0, r11, 0) // movss [r11], xmm0
	case types.DoubleType:
		a.mem(0xf2, false, []byte{0x0f, 0x11}, 0, r11, 0) // movsd [r11], xmm0
	case types.SInt8Type, types.UInt8Type:
		a.mem(0, false, []byte{0x88}, rax, r11, 0)
	case types.SInt16Type, types.UInt16Type:
		a.mem(0x66, false, []byte{0x89}, rax, r11, 0)
	case types.SInt32Type, types.UInt32Type, types.IntType:
		a.mem(0, false, []byte{0x89}, rax, r11, 0)
	default:
		a.store64(rax, r11, 0)
	}
	a.b[from-1] = byte(len(a.b) - from)
}

// epilogue returns with the errno in EAX.
func (a *asm) epilogue() {
	a.emit(0x48, 0x8d, 0x65, 0x00) // lea rsp, [rbp]
	a.emit(0x5d, 0xc3)             // pop rbp; ret
}
