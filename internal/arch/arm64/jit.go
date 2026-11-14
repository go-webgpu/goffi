// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build arm64 && (linux || darwin || windows || freebsd)

package arm64

import (
	"encoding/binary"
	"runtime"
	"unsafe"

	"github.com/go-webgpu/goffi/internal/jit"
	"github.com/go-webgpu/goffi/types"
)

// Compile implements arch.Compiler: it returns a call bridge for cif (see
// package jit), or nil when cif needs the generic call path.
func (i *Implementation) Compile(cif *types.CallInterface) unsafe.Pointer {
	code := bridgeCode(cif, runtime.GOOS)
	if code == nil {
		return nil
	}
	return unsafe.Pointer(jit.Install(code))
}

// bridgeKind reports whether a bridge passes and returns values of t's kind.
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

// Registers the bridge uses besides the argument registers: X9 holds
// &avalue[0], X10 and X11 are scratch, X15 holds the C function.
const (
	x9  = 9
	x10 = 10
	x11 = 11
	x15 = 15
	fp  = 29
	lr  = 30
	sp  = 31
)

type a64 struct{ b []byte }

func (a *a64) inst(w uint32) { a.b = binary.LittleEndian.AppendUint32(a.b, w) }

// ldst emits a load or store with an unsigned, scaled 12-bit offset: op is
// the instruction with all register and offset fields zero.
func (a *a64) ldst(op uint32, scale uint, rt, rn int, off uintptr) {
	a.inst(op | uint32(off>>scale)<<10 | uint32(rn)<<5 | uint32(rt))
}

// unscaled emits a load or store with a signed 9-bit byte offset.
func (a *a64) unscaled(op uint32, rt, rn int, off uintptr) {
	a.inst(op | (uint32(off)&0x1ff)<<12 | uint32(rn)<<5 | uint32(rt))
}

// pair emits LDP/STP with a signed 7-bit offset in 8-byte units.
func (a *a64) pair(op uint32, rt, rt2, rn int, off int) {
	a.inst(op | (uint32(off/8)&0x7f)<<15 | uint32(rt2)<<10 | uint32(rn)<<5 | uint32(rt))
}

// Load and store opcodes, immediate offset forms.
const (
	ldrsbX = 0x39800000
	ldrbW  = 0x39400000
	ldrshX = 0x79800000
	ldrhW  = 0x79400000
	ldrswX = 0xb9800000
	ldrW   = 0xb9400000
	ldrX   = 0xf9400000
	ldrS   = 0xbd400000
	ldrD   = 0xfd400000
	strbW  = 0x39000000
	strhW  = 0x79000000
	strW   = 0xb9000000
	strX   = 0xf9000000
	strS   = 0xbd000000
	strD   = 0xfd000000
	sturbW = 0x38000000
	sturhW = 0x78000000
	sturW  = 0xb8000000
	sturX  = 0xf8000000
	ldpX   = 0xa9400000
	stpX   = 0xa9000000
)

// loadInt loads the integer of kind k at [base] into Xdst, extended to 64
// bits as loadInteger extends it.
func (a *a64) loadInt(k types.TypeKind, dst, base int) {
	switch k {
	case types.SInt8Type:
		a.ldst(ldrsbX, 0, dst, base, 0)
	case types.UInt8Type:
		a.ldst(ldrbW, 0, dst, base, 0)
	case types.SInt16Type:
		a.ldst(ldrshX, 1, dst, base, 0)
	case types.UInt16Type:
		a.ldst(ldrhW, 1, dst, base, 0)
	case types.SInt32Type, types.IntType:
		a.ldst(ldrswX, 2, dst, base, 0)
	case types.UInt32Type:
		a.ldst(ldrW, 2, dst, base, 0)
	default:
		a.ldst(ldrX, 3, dst, base, 0)
	}
}

// bridgeCode returns the AAPCS64 bridge for cif on goos, or nil when an
// argument or the result is a struct. Arguments go where layoutArgs puts
// them for Execute: on Apple platforms a stack argument takes only its own
// size unless it is variadic.
//
//	stp x29, x30, [sp, #-32]!; mov x29, sp
//	[x29+16] rvalue, [x29+24] errnoFn   copied from the jit frame, which
//	                                     a Go callback may invalidate
//	[sp]     outgoing stack arguments
func bridgeCode(cif *types.CallInterface, goos string) []byte {
	ret := cif.ReturnType
	if ret == nil || (ret.Kind != types.VoidType && !bridgeKind(ret)) {
		return nil
	}
	for _, t := range cif.ArgTypes {
		if !bridgeKind(t) {
			return nil
		}
	}
	locs, stackBytes, err := layoutArgs(cif, goos)
	if err != nil || stackBytes > maxStackBytes {
		return nil
	}

	var a a64
	a.inst(0xa9be7bfd) // stp x29, x30, [sp, #-32]!
	a.inst(0x910003fd) // mov x29, sp
	a.pair(ldpX, x10, x11, 0, 16)
	a.pair(stpX, x10, x11, fp, 16)
	a.ldst(ldrX, 3, x15, 0, 0)
	a.ldst(ldrX, 3, x9, 0, 8)
	if frame := (stackBytes + 15) &^ 15; frame > 0 {
		a.inst(0xd10003ff | uint32(frame)<<10) // sub sp, sp, #frame
	}
	for i, loc := range locs {
		if loc.kind != locStack {
			continue
		}
		k := cif.ArgTypes[i].Kind
		a.ldst(ldrX, 3, x10, x9, uintptr(8*i))
		switch {
		case loc.widen:
			a.loadInt(k, x11, x10)
			a.unscaled(sturX, x11, sp, loc.offset)
		case loc.size == 1:
			a.ldst(ldrbW, 0, x11, x10, 0)
			a.unscaled(sturbW, x11, sp, loc.offset)
		case loc.size == 2:
			a.ldst(ldrhW, 1, x11, x10, 0)
			a.unscaled(sturhW, x11, sp, loc.offset)
		case loc.size == 4:
			a.ldst(ldrW, 2, x11, x10, 0)
			a.unscaled(sturW, x11, sp, loc.offset)
		default:
			a.ldst(ldrX, 3, x11, x10, 0)
			a.unscaled(sturX, x11, sp, loc.offset)
		}
	}
	for i, loc := range locs {
		k := cif.ArgTypes[i].Kind
		switch loc.kind {
		case locGPR:
			a.ldst(ldrX, 3, x10, x9, uintptr(8*i))
			a.loadInt(k, loc.reg, x10)
		case locFPR:
			a.ldst(ldrX, 3, x10, x9, uintptr(8*i))
			if k == types.FloatType {
				a.ldst(ldrS, 2, loc.reg, x10, 0)
			} else {
				a.ldst(ldrD, 3, loc.reg, x10, 0)
			}
		}
	}
	a.inst(0xd63f0000 | x15<<5) // blr x15
	a.inst(0x910003bf)          // mov sp, x29
	a.pair(ldpX, x10, x11, sp, 16)

	if ret.Kind != types.VoidType {
		a.inst(0xb4000000 | 2<<5 | x10) // cbz x10, past the store
		switch ret.Kind {
		case types.FloatType:
			a.ldst(strS, 2, 0, x10, 0)
		case types.DoubleType:
			a.ldst(strD, 3, 0, x10, 0)
		case types.SInt8Type, types.UInt8Type:
			a.ldst(strbW, 0, 0, x10, 0)
		case types.SInt16Type, types.UInt16Type:
			a.ldst(strhW, 1, 0, x10, 0)
		case types.SInt32Type, types.UInt32Type, types.IntType:
			a.ldst(strW, 2, 0, x10, 0)
		default:
			a.ldst(strX, 3, 0, x10, 0)
		}
	}
	a.inst(0x52800000)              // mov w0, #0
	a.inst(0xb4000000 | 3<<5 | x11) // cbz x11, past the errno read
	a.inst(0xd63f0000 | x11<<5)     // blr x11
	a.ldst(ldrW, 2, 0, 0, 0)        // ldr w0, [x0]
	a.inst(0xa8c27bfd)              // ldp x29, x30, [sp], #32
	a.inst(0xd65f03c0)              // ret
	return a.b
}
