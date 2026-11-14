// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build arm64 && (linux || darwin || windows || freebsd)

package arm64

import (
	"bytes"
	"math/rand/v2"
	"runtime"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// Written by probe (jit_probe_test.s).
var (
	probeRegs  [24]uint64 // X0-X7, D0-D7, 8 stack words
	probeRet   uint64
	probeErrno int32
)

const (
	probeFPR   = 8
	probeStack = 16
)

func probeAddrs() (probe, errno uintptr)

var bridgeTypes = []*types.TypeDescriptor{
	types.SInt8TypeDescriptor, types.UInt8TypeDescriptor,
	types.SInt16TypeDescriptor, types.UInt16TypeDescriptor,
	types.SInt32TypeDescriptor, types.UInt32TypeDescriptor,
	types.SInt64TypeDescriptor, types.UInt64TypeDescriptor,
	{Size: 4, Alignment: 4, Kind: types.IntType},
	types.PointerTypeDescriptor, types.FloatTypeDescriptor, types.DoubleTypeDescriptor,
}

type probeResult struct {
	regs  [24]uint64
	ret   [16]byte
	errno uintptr
}

func probeCall(t *testing.T, cif *types.CallInterface, avalue []unsafe.Pointer, nilResult bool) probeResult {
	t.Helper()
	fn, errnoFn := probeAddrs()
	var r probeResult
	for i := range r.ret {
		r.ret[i] = 0xaa
	}
	rvalue := unsafe.Pointer(&r.ret[0])
	if nilResult {
		rvalue = nil
	}
	probeRegs = [24]uint64{}
	errno, err := (&Implementation{}).Execute(cif, *(*unsafe.Pointer)(unsafe.Pointer(&fn)), rvalue, avalue, errnoFn)
	if err != nil {
		t.Fatal(err)
	}
	r.regs, r.errno = probeRegs, errno
	return r
}

// TestBridgeMatchesExecute calls the probe with random scalar signatures,
// variadic ones included, and random values through a bridge and through
// the generic path, and checks that the callee receives the same bits in
// every register and stack byte that holds an argument, and that the
// caller gets the same result bytes and errno back.
func TestBridgeMatchesExecute(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	rets := append([]*types.TypeDescriptor{types.VoidTypeDescriptor}, bridgeTypes...)
	compiled := 0
	for iter := range 3000 {
		n := rng.IntN(24)
		cif := &types.CallInterface{ReturnType: rets[rng.IntN(len(rets))], ArgTypes: make([]*types.TypeDescriptor, n)}
		for i := range n {
			cif.ArgTypes[i] = bridgeTypes[rng.IntN(len(bridgeTypes))]
		}
		if n > 0 && iter%4 == 0 {
			cif.FixedArgCount = 1 + rng.IntN(n)
		}
		cif.Flags = classifyReturnARM64(cif.ReturnType, types.DefaultConvention())
		locs, stackBytes, err := layoutArgs(cif, runtime.GOOS)
		if err != nil || stackBytes > maxStackBytes {
			continue
		}
		bridge := (&Implementation{}).Compile(cif)
		if bridge == nil {
			t.Skip("no bridges on this platform")
		}
		compiled++

		vals := make([]uint64, n)
		avalue := make([]unsafe.Pointer, n)
		for i := range vals {
			vals[i] = rng.Uint64()
			avalue[i] = unsafe.Pointer(&vals[i])
		}
		// errno values are positive; the generic arm64 path sign-extends the
		// int it reads, the amd64 path and the bridges zero-extend it.
		probeRet, probeErrno = rng.Uint64(), rng.Int32()
		nilResult := iter%7 == 0

		generic := probeCall(t, cif, avalue, nilResult)
		withBridge := *cif
		withBridge.Bridge = bridge
		got := probeCall(t, &withBridge, avalue, nilResult)

		gotStack := unsafe.Slice((*byte)(unsafe.Pointer(&got.regs[probeStack])), 64)
		genericStack := unsafe.Slice((*byte)(unsafe.Pointer(&generic.regs[probeStack])), 64)
		for i, loc := range locs {
			var g, w []byte
			switch loc.kind {
			case locGPR:
				g, w = u64bytes(&got.regs[loc.reg]), u64bytes(&generic.regs[loc.reg])
			case locFPR:
				g, w = u64bytes(&got.regs[probeFPR+loc.reg]), u64bytes(&generic.regs[probeFPR+loc.reg])
			case locStack:
				size := loc.size
				if loc.widen {
					size = 8
				}
				g, w = gotStack[loc.offset:loc.offset+size], genericStack[loc.offset:loc.offset+size]
			}
			if !bytes.Equal(g, w) {
				t.Fatalf("iteration %d, %v fixed %d: argument %d (kind %d) at %+v: bridge passed %x, Execute %x",
					iter, kinds(cif), cif.FixedArgCount, i, cif.ArgTypes[i].Kind, loc, g, w)
			}
		}
		if !bytes.Equal(got.ret[:], generic.ret[:]) {
			t.Fatalf("iteration %d, %v: result bytes %x, Execute wrote %x", iter, kinds(cif), got.ret, generic.ret)
		}
		if got.errno != generic.errno {
			t.Fatalf("iteration %d: errno %d, Execute %d", iter, got.errno, generic.errno)
		}
	}
	if compiled < 1000 {
		t.Fatalf("only %d signatures compiled", compiled)
	}
}

func u64bytes(p *uint64) []byte { return unsafe.Slice((*byte)(unsafe.Pointer(p)), 8) }

func kinds(cif *types.CallInterface) []types.TypeKind {
	k := []types.TypeKind{cif.ReturnType.Kind}
	for _, t := range cif.ArgTypes {
		k = append(k, t.Kind)
	}
	return k
}
