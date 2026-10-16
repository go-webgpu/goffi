// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build arm64

package arm64

import (
	"testing"

	"github.com/go-webgpu/goffi/types"
)

// The expected locations follow libffi's aarch64 ffi_call_int and
// allocate_to_stack, and Apple's "Writing ARM64 code for Apple platforms".
// layoutArgs takes the OS as a parameter, so both variants run on any arm64.

func repeatTD(td *types.TypeDescriptor, n int) []*types.TypeDescriptor {
	out := make([]*types.TypeDescriptor, n)
	for i := range out {
		out[i] = td
	}
	return out
}

func structTD(members ...*types.TypeDescriptor) *types.TypeDescriptor {
	return &types.TypeDescriptor{Kind: types.StructType, Members: members}
}

func mustLayout(t *testing.T, goos string, fixed int, args ...*types.TypeDescriptor) ([]argLoc, uintptr) {
	t.Helper()
	cif := &types.CallInterface{ArgTypes: args, ArgCount: len(args), FixedArgCount: fixed}
	locs, bytes, err := layoutArgs(cif, goos)
	if err != nil {
		t.Fatalf("layoutArgs(%s): %v", goos, err)
	}
	return locs, bytes
}

func wantStack(t *testing.T, loc argLoc, offset, size uintptr, widen bool) {
	t.Helper()
	if loc.kind != locStack || loc.offset != offset || loc.size != size || loc.widen != widen {
		t.Errorf("got %+v, want stack offset %d size %d widen %v", loc, offset, size, widen)
	}
}

func wantReg(t *testing.T, loc argLoc, kind locKind, reg, nregs int) {
	t.Helper()
	if loc.kind != kind || loc.reg != reg || loc.nregs != nregs {
		t.Errorf("got %+v, want kind %d reg %d nregs %d", loc, kind, reg, nregs)
	}
}

func TestLayoutInt32StackArgs(t *testing.T) {
	args := repeatTD(types.SInt32TypeDescriptor, 10)

	locs, bytes := mustLayout(t, "linux", 0, args...)
	for i := 0; i < 8; i++ {
		wantReg(t, locs[i], locGPR, i, 1)
	}
	wantStack(t, locs[8], 0, 4, true)
	wantStack(t, locs[9], 8, 4, true)
	if bytes != 16 {
		t.Errorf("linux stack bytes = %d, want 16", bytes)
	}

	// Apple packs the two int32 into one 8-byte word: [sp] and [sp+4].
	locs, bytes = mustLayout(t, "darwin", 0, args...)
	wantStack(t, locs[8], 0, 4, false)
	wantStack(t, locs[9], 4, 4, false)
	if bytes != 8 {
		t.Errorf("darwin stack bytes = %d, want 8", bytes)
	}
}

func TestLayoutFloatStackArgs(t *testing.T) {
	args := repeatTD(types.FloatTypeDescriptor, 10)

	locs, _ := mustLayout(t, "linux", 0, args...)
	for i := 0; i < 8; i++ {
		wantReg(t, locs[i], locFPR, i, 1)
	}
	wantStack(t, locs[8], 0, 4, false)
	wantStack(t, locs[9], 8, 4, false)

	locs, _ = mustLayout(t, "darwin", 0, args...)
	wantStack(t, locs[8], 0, 4, false)
	wantStack(t, locs[9], 4, 4, false)
}

func TestLayoutNarrowIntegerMix(t *testing.T) {
	args := append(repeatTD(types.SInt64TypeDescriptor, 8),
		types.SInt8TypeDescriptor, types.SInt16TypeDescriptor, types.SInt32TypeDescriptor, types.SInt8TypeDescriptor)

	locs, bytes := mustLayout(t, "linux", 0, args...)
	wantStack(t, locs[8], 0, 1, true)
	wantStack(t, locs[9], 8, 2, true)
	wantStack(t, locs[10], 16, 4, true)
	wantStack(t, locs[11], 24, 1, true)
	if bytes != 32 {
		t.Errorf("linux stack bytes = %d, want 32", bytes)
	}

	// Apple: natural alignment, own size: 0, 2, 4, 8 -> 9 bytes, rounded to 16.
	locs, bytes = mustLayout(t, "darwin", 0, args...)
	wantStack(t, locs[8], 0, 1, false)
	wantStack(t, locs[9], 2, 2, false)
	wantStack(t, locs[10], 4, 4, false)
	wantStack(t, locs[11], 8, 1, false)
	if bytes != 16 {
		t.Errorf("darwin stack bytes = %d, want 16", bytes)
	}
}

func TestLayoutHFAOverflowGoesToStackWhole(t *testing.T) {
	hfa := structTD(types.FloatTypeDescriptor, types.FloatTypeDescriptor)
	args := append(repeatTD(types.FloatTypeDescriptor, 7), hfa, types.FloatTypeDescriptor)

	for _, goos := range []string{"linux", "darwin"} {
		locs, _ := mustLayout(t, goos, 0, args...)
		// One V register is left, the HFA needs two: it goes to the stack as
		// one 8-byte block, and no more V registers are used afterwards.
		wantStack(t, locs[7], 0, 8, false)
		if locs[8].kind != locStack {
			t.Errorf("%s: float after an overflowed HFA = %+v, want stack", goos, locs[8])
		}
	}
}

func TestLayoutHFAInRegisters(t *testing.T) {
	cgRect := structTD(
		structTD(types.DoubleTypeDescriptor, types.DoubleTypeDescriptor),
		structTD(types.DoubleTypeDescriptor, types.DoubleTypeDescriptor),
	)
	locs, _ := mustLayout(t, "darwin", 0, types.PointerTypeDescriptor, cgRect)
	wantReg(t, locs[0], locGPR, 0, 1)
	wantReg(t, locs[1], locFPR, 0, 4)
	if locs[1].elem != 8 {
		t.Errorf("nested double HFA element size = %d, want 8", locs[1].elem)
	}
}

func TestLayoutMixedSmallStructUsesXRegisters(t *testing.T) {
	mixed := structTD(types.SInt64TypeDescriptor, types.DoubleTypeDescriptor)
	locs, _ := mustLayout(t, "linux", 0, mixed, types.DoubleTypeDescriptor)
	wantReg(t, locs[0], locGPR, 0, 2)
	wantReg(t, locs[1], locFPR, 0, 1) // the double member did not use V0
}

func TestLayoutSmallStructOverflowStopsXRegisters(t *testing.T) {
	pair := structTD(types.SInt64TypeDescriptor, types.SInt64TypeDescriptor)
	args := append(repeatTD(types.SInt64TypeDescriptor, 7), pair, types.SInt64TypeDescriptor)

	locs, _ := mustLayout(t, "linux", 0, args...)
	// X7 is free but the struct needs two registers: it is copied to the
	// stack, and the int64 after it goes to the stack too (NGRN = 8).
	wantStack(t, locs[7], 0, 16, false)
	wantStack(t, locs[8], 16, 8, true)
}

func TestLayoutLargeStructByReference(t *testing.T) {
	large := structTD(types.UInt64TypeDescriptor, types.UInt64TypeDescriptor, types.UInt64TypeDescriptor)
	locs, _ := mustLayout(t, "linux", 0, large)
	if locs[0].kind != locGPR || !locs[0].byRef || locs[0].reg != 0 {
		t.Errorf("24-byte struct = %+v, want a pointer in X0", locs[0])
	}

	args := append(repeatTD(types.SInt64TypeDescriptor, 8), large)
	locs, _ = mustLayout(t, "darwin", 0, args...)
	if locs[8].kind != locStack || !locs[8].byRef || locs[8].offset != 0 || locs[8].size != 8 {
		t.Errorf("24-byte struct after 8 int64 = %+v, want a pointer at [sp]", locs[8])
	}
}

func TestLayoutAppleVariadicUsesEightByteSlots(t *testing.T) {
	// printf-like: (const char *fmt, ...) called with an int32 and a double.
	args := []*types.TypeDescriptor{types.PointerTypeDescriptor, types.SInt32TypeDescriptor, types.DoubleTypeDescriptor}

	locs, _ := mustLayout(t, "darwin", 1, args...)
	wantReg(t, locs[0], locGPR, 0, 1)
	wantStack(t, locs[1], 0, 4, true)
	wantStack(t, locs[2], 8, 8, false)

	// Linux passes variadic arguments like any other.
	locs, _ = mustLayout(t, "linux", 1, args...)
	wantReg(t, locs[1], locGPR, 1, 1)
	wantReg(t, locs[2], locFPR, 0, 1)
}

func TestLayoutStackOverflowIsReported(t *testing.T) {
	// Sixteen doubles: eight in V registers, eight 8-byte words on the stack,
	// one more than the call path copies.
	cif := &types.CallInterface{ArgTypes: repeatTD(types.DoubleTypeDescriptor, 16), ArgCount: 16}
	need, limit, err := (&Implementation{}).StackArgBytes(cif)
	if err != nil {
		t.Fatal(err)
	}
	if need != 64 || limit != 56 {
		t.Errorf("StackArgBytes = (%d, %d), want (64, 56)", need, limit)
	}
}
