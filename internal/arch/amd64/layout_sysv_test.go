// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build amd64 && (linux || darwin || freebsd)

package amd64

import (
	"math/rand/v2"
	"testing"

	"github.com/go-webgpu/goffi/types"
)

// TestPlaceScalarMatchesPlace checks that the fast path for non-struct
// arguments assigns the same registers and stack words as place, and leaves
// the allocator in the same state, whatever came before it.
func TestPlaceScalarMatchesPlace(t *testing.T) {
	pool := []*types.TypeDescriptor{
		types.IntTypeDescriptor, types.FloatTypeDescriptor, types.DoubleTypeDescriptor,
		types.UInt8TypeDescriptor, types.SInt8TypeDescriptor,
		types.UInt16TypeDescriptor, types.SInt16TypeDescriptor,
		types.UInt32TypeDescriptor, types.SInt32TypeDescriptor,
		types.UInt64TypeDescriptor, types.SInt64TypeDescriptor,
		types.PointerTypeDescriptor,
		{Size: 8, Alignment: 16, Kind: types.SInt64Type},
		{Size: 8, Alignment: 16, Kind: types.DoubleType},
		{Size: 16, Alignment: 8, Kind: types.StructType, Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor, types.DoubleTypeDescriptor}},
		{Size: 16, Alignment: 8, Kind: types.StructType, Members: []*types.TypeDescriptor{
			types.SInt64TypeDescriptor, types.DoubleTypeDescriptor}},
		{Size: 24, Alignment: 8, Kind: types.StructType, Members: []*types.TypeDescriptor{
			types.SInt64TypeDescriptor, types.SInt64TypeDescriptor, types.SInt64TypeDescriptor}},
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for range 2000 {
		var fast, slow SysVAllocator
		if rng.IntN(4) == 0 {
			fast.GPR()
			slow.GPR()
		}
		for i := range 1 + rng.IntN(24) {
			typ := pool[rng.IntN(len(pool))]
			want := slow.place(typ)
			if typ.Kind == types.StructType {
				fast.place(typ)
				continue
			}
			reg, sse, word := fast.placeScalar(typ)
			got := sysvPlacement{onStack: reg < 0, word: word}
			got.sse[0] = sse
			if reg >= 0 {
				got.n, got.reg[0], got.word = 1, reg, 0
			} else {
				got.sse[0] = false
			}
			if want.onStack {
				want.sse[0] = false
			}
			if got != want || fast != slow {
				t.Fatalf("argument %d (%+v): placeScalar = %+v, allocator %+v; place = %+v, allocator %+v",
					i, *typ, got, fast, want, slow)
			}
		}
	}
}
