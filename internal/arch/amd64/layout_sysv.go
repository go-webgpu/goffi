// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build amd64 && (linux || darwin || freebsd)

package amd64

import (
	"github.com/go-webgpu/goffi/types"
)

// Argument layout for the System V AMD64 psABI (§3.2.3, "Parameter Passing"),
// as libffi implements it (src/x86/ffi64.c: ffi_prep_cif_machdep sizes the
// stack, ffi_call_int places the values):
//
//   - Integers and pointers take one INTEGER eightbyte, floats and doubles
//     one SSE eightbyte. A struct of at most 16 bytes is split into
//     eightbytes, each INTEGER or SSE by its members (INTEGER wins); a larger
//     struct is MEMORY.
//   - If every eightbyte of an argument gets a register of its class
//     (RDI, RSI, RDX, RCX, R8, R9 / XMM0-XMM7), the argument goes in
//     registers. Otherwise the whole argument goes on the stack, and the
//     registers that were left stay free for later arguments.
//   - Stack arguments are aligned to at least 8 bytes and take whole
//     8-byte words, in argument order.

const (
	sysvGPRs = 6 // RDI, RSI, RDX, RCX, R8, R9
	sysvSSEs = 8 // XMM0-XMM7

	// MaxStackBytes bounds the stack area for one C call. The words are
	// copied onto the system stack (g0) the call runs on, which is the OS
	// thread's stack: 512 KiB for a secondary thread on macOS, 8 MiB by
	// default on Linux. libffi has no bound (it uses alloca); 64 KiB leaves
	// that stack to the callee, and no real C signature comes close (8192
	// words, e.g. 680 structs of 96 bytes).
	MaxStackBytes = 64 << 10
)

// SysVAllocator hands out registers and stack words to arguments in the order
// they are passed. The ffi package uses it to decode callback arguments with
// the same rules the call path places them.
type SysVAllocator struct {
	gpr, sse int
	stack    uintptr // stack bytes used so far
}

// Fits reports whether ngpr integer and nsse SSE registers are still free.
func (a *SysVAllocator) Fits(ngpr, nsse int) bool {
	return a.gpr+ngpr <= sysvGPRs && a.sse+nsse <= sysvSSEs
}

// GPR takes the next integer register and returns its index (0 = RDI).
func (a *SysVAllocator) GPR() int {
	r := a.gpr
	a.gpr++
	return r
}

// SSE takes the next SSE register and returns its index (0 = XMM0).
func (a *SysVAllocator) SSE() int {
	r := a.sse
	a.sse++
	return r
}

// Stack reserves size bytes, aligned to at least 8, and returns the index of
// the first 8-byte word.
func (a *SysVAllocator) Stack(size, align uintptr) int {
	if align < 8 {
		align = 8
	}
	a.stack = (a.stack + align - 1) &^ (align - 1)
	word := int(a.stack / 8)
	a.stack += (size + 7) &^ 7
	return word
}

// StackBytes returns the stack bytes reserved so far (a multiple of 8).
func (a *SysVAllocator) StackBytes() uintptr {
	return a.stack
}

// sysvPlacement says where one argument goes.
type sysvPlacement struct {
	n       int     // eightbytes passed in registers (0 when on the stack)
	sse     [2]bool // class of each eightbyte: SSE (true) or INTEGER
	reg     [2]int  // register index of each eightbyte
	onStack bool
	word    int // first stack word when onStack
}

// sysvClasses classifies t: n eightbytes with the given classes, or memory
// when t is passed on the stack whatever registers are free. A zero-size
// struct has no eightbytes and is not passed at all.
func sysvClasses(t *types.TypeDescriptor) (sse [2]bool, n int, memory bool) {
	switch t.Kind {
	case types.FloatType, types.DoubleType:
		return [2]bool{true, false}, 1, false
	case types.StructType:
		if t.Size > 16 {
			return sse, 0, true
		}
		n = int((t.Size + 7) / 8)
		for i := range n {
			start := uintptr(i) * 8
			sse[i] = classifyEightbyte(t, start, min(start+8, t.Size))
		}
		return sse, n, false
	default:
		return sse, 1, false
	}
}

// placeScalar is place for a type that is not a struct. Such an argument is one
// eightbyte: an SSE register for float and double, an integer register
// otherwise, or one stack word once that class has run out. reg is -1 when the
// argument is on the stack, at word.
func (a *SysVAllocator) placeScalar(t *types.TypeDescriptor) (reg int, sse bool, word int) {
	sse = t.Kind == types.FloatType || t.Kind == types.DoubleType
	switch {
	case sse && a.sse < sysvSSEs:
		return a.SSE(), true, 0
	case !sse && a.gpr < sysvGPRs:
		return a.GPR(), false, 0
	}
	return -1, sse, a.Stack(8, t.Alignment)
}

// place assigns t its registers or stack words.
func (a *SysVAllocator) place(t *types.TypeDescriptor) sysvPlacement {
	sse, n, memory := sysvClasses(t)
	var p sysvPlacement
	if !memory {
		ngpr, nsse := 0, 0
		for i := range n {
			if sse[i] {
				nsse++
			} else {
				ngpr++
			}
		}
		if a.Fits(ngpr, nsse) {
			p.n, p.sse = n, sse
			for i := range n {
				if sse[i] {
					p.reg[i] = a.SSE()
				} else {
					p.reg[i] = a.GPR()
				}
			}
			return p
		}
	}
	size := t.Size
	if t.Kind != types.StructType {
		size = 8
	}
	if size == 0 {
		return p
	}
	p.onStack = true
	p.word = a.Stack(size, t.Alignment)
	return p
}

// sysvRetInMemory reports whether the result of cif is returned through a
// hidden pointer, which takes RDI before any argument.
func sysvRetInMemory(cif *types.CallInterface) bool {
	return cif.ReturnType.Kind == types.StructType && cif.ReturnType.Size > 16
}

// sysvStackBytes returns the stack bytes the arguments of cif need.
func sysvStackBytes(cif *types.CallInterface) uintptr {
	var a SysVAllocator
	if sysvRetInMemory(cif) {
		a.GPR()
	}
	for _, t := range cif.ArgTypes {
		if t.Kind == types.StructType {
			a.place(t)
		} else {
			a.placeScalar(t)
		}
	}
	return a.StackBytes()
}

// StackArgBytes reports how many bytes of stack the arguments of cif need and
// how many the call path supports. PrepareCallInterface uses it to reject
// calls that do not fit.
func (i *Implementation) StackArgBytes(cif *types.CallInterface) (need, limit uintptr, err error) {
	return sysvStackBytes(cif), MaxStackBytes, nil
}
