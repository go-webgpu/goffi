// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build ((linux && !android) || darwin || freebsd) && (amd64 || arm64)

package ffi

import (
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// TestOutPointerSurvivesStackMove checks the contract of CallFunctionContext's
// documentation: memory reachable from avalue through pointer-typed values
// stays in place for the call. C calls back into Go, the callback grows (and
// so moves) the goroutine stack, then C writes through the out-pointer to the
// caller's local. Held as an unsafe.Pointer, the local is heap memory and
// receives the value.
//
// There is no negative control with a uintptr: then the local can stay on the
// stack and C writes into the old, freed stack. That corrupts memory instead
// of failing cleanly, so it cannot be a test.
func TestOutPointerSurvivesStackMove(t *testing.T) {
	requireStructLib(t)
	sym, err := GetSymbol(structTestLib, "call_then_store")
	if err != nil {
		t.Fatal(err)
	}
	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.PointerTypeDescriptor, types.SInt64TypeDescriptor},
	); err != nil {
		t.Fatal(err)
	}
	cb := NewCallback(func() {
		// About 2 MiB of stack growth, enough to force a copystack mid-call.
		growStack(512)
	})

	done := make(chan int64, 1)
	go func() {
		// A fresh goroutine starts with a small stack, so the growth moves it.
		var out int64
		outPtr := unsafe.Pointer(&out)
		v := int64(0x5eed_cafe)
		if _, err := CallFunction(&cif, sym, nil,
			[]unsafe.Pointer{unsafe.Pointer(&cb), unsafe.Pointer(&outPtr), unsafe.Pointer(&v)}); err != nil {
			t.Error(err)
		}
		done <- out
	}()

	if got := <-done; got != 0x5eed_cafe {
		t.Fatalf("out = %#x after the call, want 0x5eedcafe (C wrote to a moved address)", got)
	}
}
