// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build amd64 && (linux || darwin || freebsd || windows)

package amd64

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
	probeRegs  [31]uint64 // RDI RSI RDX RCX R8 R9 RAX, XMM0-7, stack words
	probeRet   uint64
	probeErrno int32
)

const (
	probeRAX   = 6
	probeXMM   = 7
	probeStack = 15
)

func probeAddrs() (probe, errno uintptr)

var probeGPR = map[int]int{rdi: 0, rsi: 1, rdx: 2, rcx: 3, r8: 4, r9: 5}

var bridgeTypes = []*types.TypeDescriptor{
	types.SInt8TypeDescriptor, types.UInt8TypeDescriptor,
	types.SInt16TypeDescriptor, types.UInt16TypeDescriptor,
	types.SInt32TypeDescriptor, types.UInt32TypeDescriptor,
	types.SInt64TypeDescriptor, types.UInt64TypeDescriptor,
	{Size: 4, Alignment: 4, Kind: types.IntType},
	types.PointerTypeDescriptor, types.FloatTypeDescriptor, types.DoubleTypeDescriptor,
}

type probeResult struct {
	regs  [31]uint64
	ret   [16]byte
	errno uintptr
}

// probeCall calls the probe through Execute, through the bridge when cif
// has one, and returns what the probe saw and what the caller got back.
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
	probeRegs = [31]uint64{}
	errno, err := (&Implementation{}).Execute(cif, *(*unsafe.Pointer)(unsafe.Pointer(&fn)), rvalue, avalue, errnoFn)
	if err != nil {
		t.Fatal(err)
	}
	r.regs, r.errno = probeRegs, errno
	return r
}

// TestBridgeMatchesExecute calls the probe with random scalar signatures
// and values through a bridge and through the generic path, and checks
// that the callee receives the same bits in every register and stack word
// that holds an argument, and that the caller gets the same result bytes
// and errno back.
func TestBridgeMatchesExecute(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	rets := append([]*types.TypeDescriptor{types.VoidTypeDescriptor}, bridgeTypes...)
	compiled := 0
	for iter := range 3000 {
		n := rng.IntN(probeMaxArgs + 1)
		cif := &types.CallInterface{ReturnType: rets[rng.IntN(len(rets))], ArgTypes: make([]*types.TypeDescriptor, n)}
		for i := range n {
			cif.ArgTypes[i] = bridgeTypes[rng.IntN(len(bridgeTypes))]
		}
		cif.Flags = classifyReturnAMD64(cif.ReturnType, types.DefaultConvention())
		locs, nsse, fits := probeLocations(cif)
		if !fits {
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

		for i, loc := range locs {
			for _, slot := range loc {
				if got.regs[slot] != generic.regs[slot] {
					t.Fatalf("iteration %d, %v: argument %d (kind %d) in probe slot %d: bridge passed %#x, Execute %#x",
						iter, kinds(cif), i, cif.ArgTypes[i].Kind, slot, got.regs[slot], generic.regs[slot])
				}
			}
		}
		if runtime.GOOS != "windows" && got.regs[probeRAX]&0xff != uint64(nsse) {
			t.Fatalf("iteration %d: AL = %d, want %d vector registers", iter, got.regs[probeRAX]&0xff, nsse)
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

func kinds(cif *types.CallInterface) []types.TypeKind {
	k := []types.TypeKind{cif.ReturnType.Kind}
	for _, t := range cif.ArgTypes {
		k = append(k, t.Kind)
	}
	return k
}
