// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build arm64

package arm64

import (
	"fmt"

	"github.com/go-webgpu/goffi/types"
)

// Argument layout for AAPCS64 and Apple's arm64 variant.
//
// The rules follow the Arm Procedure Call Standard (AAPCS64) parameter
// passing stage C, as implemented by libffi (src/aarch64/ffi.c,
// ffi_call_int and allocate_to_stack), plus the deviations in Apple's
// "Writing ARM64 code for Apple platforms":
//
//   - Integers and pointers use X0-X7, floats and doubles V0-V7. Once a class
//     runs out of registers its arguments go on the stack.
//   - An HFA (one to four floats or doubles, possibly nested) uses consecutive
//     V registers if all of them are free; otherwise no more V registers are
//     used and the whole HFA is copied to the stack.
//   - Any other composite of at most 16 bytes is copied, as raw bytes, into
//     consecutive X registers if enough are free (floating-point members
//     included); otherwise no more X registers are used and it is copied to
//     the stack.
//   - A larger composite is copied to memory owned by the caller and passed
//     as a pointer to the copy.
//   - Standard AAPCS64 aligns every stack argument to at least 8 bytes and
//     widens integers to 64 bits. Apple aligns a stack argument only to its
//     natural alignment and stores exactly its size ("function arguments may
//     consume slots on the stack that are not multiples of 8 bytes"), except
//     variadic arguments, which take whole 8-byte slots. On Apple platforms
//     every variadic argument goes on the stack.

// maxStackBytes is the stack area the call trampoline copies for a C call:
// maxStackArgs 8-byte words.
const maxStackBytes = maxStackArgs * 8

// locKind says where an argument is passed.
type locKind uint8

const (
	locGPR   locKind = iota // X registers reg..reg+nregs-1
	locFPR                  // V registers reg..reg+nregs-1, one element each
	locStack                // stack bytes offset..offset+size-1
)

// argLoc describes where one argument goes.
type argLoc struct {
	kind   locKind
	reg    int     // first register (locGPR, locFPR)
	nregs  int     // registers used (locGPR, locFPR)
	elem   uintptr // locFPR: size of one element, 4 (float) or 8 (double)
	offset uintptr // locStack: offset from the stack pointer at the call
	size   uintptr // bytes of the value (or 8 for a pointer to a copy)
	byRef  bool    // a composite > 16 bytes: a pointer to a copy is passed
	widen  bool    // locStack integer: store sign/zero-extended to 8 bytes
}

// StackAllocator hands out stack offsets for arguments in the order they are
// passed. The ffi package uses it to find callback arguments on the stack.
type StackAllocator struct {
	next     uintptr // next stack offset (NSAA)
	apple    bool
	variadic bool
}

// NewStackAllocator returns an allocator for goos ("darwin" and "ios" use
// Apple's packing, every other OS standard AAPCS64).
func NewStackAllocator(goos string) StackAllocator {
	return StackAllocator{apple: isAppleOS(goos)}
}

// Alloc reserves size bytes for an argument with the given natural alignment
// and returns its offset from the stack pointer at the call.
func (s *StackAllocator) Alloc(align, size uintptr) uintptr {
	if align == 0 {
		align = 1
	}
	if (!s.apple || s.variadic) && align < 8 {
		align = 8
	}
	off := alignOffset(s.next, align)
	s.next = off + size
	return off
}

// Bytes returns the stack space used so far, rounded up to 8 bytes.
func (s *StackAllocator) Bytes() uintptr {
	return alignOffset(s.next, 8)
}

// WidensIntegers reports whether integer stack arguments are stored as
// 8-byte values (standard AAPCS64, Apple variadic) rather than at their own
// size (Apple non-variadic).
func (s *StackAllocator) WidensIntegers() bool {
	return !s.apple || s.variadic
}

func isAppleOS(goos string) bool {
	return goos == "darwin" || goos == "ios"
}

// layoutArgs computes where every argument of cif goes on goos. It depends
// only on the argument types, so PrepareCallInterface can validate a call
// and Execute places the values with the same rules.
func layoutArgs(cif *types.CallInterface, goos string) ([]argLoc, uintptr, error) {
	const nRegs = 8 // X0-X7 and V0-V7

	locs := make([]argLoc, len(cif.ArgTypes))
	stack := NewStackAllocator(goos)
	ngrn, nsrn := 0, 0 // next X / V register

	toStack := func(align, size uintptr) argLoc {
		return argLoc{kind: locStack, offset: stack.Alloc(align, size), size: size}
	}

	for i, t := range cif.ArgTypes {
		if t == nil {
			return nil, 0, fmt.Errorf("goffi: argument %d has no type", i)
		}
		// Apple: from the first variadic argument on, everything goes on the
		// stack in 8-byte slots.
		if stack.apple && cif.FixedArgCount > 0 && i == cif.FixedArgCount {
			ngrn, nsrn = nRegs, nRegs
			stack.variadic = true
		}

		size, align := ensureStructLayout(t)

		switch t.Kind {
		case types.FloatType, types.DoubleType:
			if nsrn < nRegs {
				locs[i] = argLoc{kind: locFPR, reg: nsrn, nregs: 1, elem: size, size: size}
				nsrn++
			} else {
				locs[i] = toStack(align, size)
			}

		case types.IntType, types.SInt8Type, types.UInt8Type, types.SInt16Type, types.UInt16Type,
			types.SInt32Type, types.UInt32Type, types.SInt64Type, types.UInt64Type, types.PointerType:
			if ngrn < nRegs {
				locs[i] = argLoc{kind: locGPR, reg: ngrn, nregs: 1, size: size}
				ngrn++
			} else {
				locs[i] = toStack(align, size)
				locs[i].widen = stack.WidensIntegers()
			}

		case types.StructType:
			if isHFA, count, elemKind := isHomogeneousFloatAggregate(t); isHFA && count <= 4 {
				elem := uintptr(8)
				if elemKind == types.FloatType {
					elem = 4
				}
				if nsrn+count <= nRegs {
					locs[i] = argLoc{kind: locFPR, reg: nsrn, nregs: count, elem: elem, size: size}
					nsrn += count
				} else {
					nsrn = nRegs
					locs[i] = toStack(align, size)
				}
				break
			}

			if size > 16 {
				// Passed as a pointer to a copy, like a pointer argument.
				if ngrn < nRegs {
					locs[i] = argLoc{kind: locGPR, reg: ngrn, nregs: 1, size: 8, byRef: true}
					ngrn++
				} else {
					locs[i] = toStack(8, 8)
					locs[i].byRef = true
				}
				break
			}

			n := int((size + 7) / 8)
			if ngrn+n <= nRegs {
				locs[i] = argLoc{kind: locGPR, reg: ngrn, nregs: n, size: size}
				ngrn += n
			} else {
				ngrn = nRegs
				locs[i] = toStack(align, size)
			}

		default:
			return nil, 0, fmt.Errorf("goffi: argument %d: unsupported type kind %d", i, t.Kind)
		}
	}

	return locs, stack.Bytes(), nil
}
