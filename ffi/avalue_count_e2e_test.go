// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux || darwin || freebsd || windows) && (amd64 || arm64)

package ffi

import (
	"errors"
	"fmt"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// avalue must hold exactly one value per argument type. Fewer must be an error
// before the call, never a call with whatever happens to be in the missing
// registers and never an index-out-of-range panic; more means the caller and
// the cif disagree about the signature.
func TestCallFunction_ArgValueCountMismatch(t *testing.T) {
	requireStructLib(t)
	sym, err := GetSymbol(structTestLib, "int_weighted3")
	if err != nil {
		t.Fatal(err)
	}
	var cif types.CallInterface
	if err = PrepareCallInterface(&cif, types.DefaultCall, types.SInt32TypeDescriptor,
		repeatType(types.SInt32TypeDescriptor, 3)); err != nil {
		t.Fatal(err)
	}
	vals := [4]int32{1, 2, 3, 4}
	ptrs := []unsafe.Pointer{
		unsafe.Pointer(&vals[0]), unsafe.Pointer(&vals[1]), unsafe.Pointer(&vals[2]), unsafe.Pointer(&vals[3]),
	}

	for _, n := range []int{0, 2, 4} {
		t.Run(fmt.Sprintf("%d values for 3 arguments", n), func(t *testing.T) {
			got := int32(-1)
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("CallFunction panicked: %v", r)
				}
			}()
			_, callErr := CallFunction(&cif, sym, unsafe.Pointer(&got), ptrs[:n])
			var icErr *InvalidCallInterfaceError
			if !errors.As(callErr, &icErr) {
				t.Fatalf("err = %v (result %d), want *InvalidCallInterfaceError", callErr, got)
			}
			if icErr.Field != "avalue" {
				t.Errorf("InvalidCallInterfaceError.Field = %q, want %q", icErr.Field, "avalue")
			}
			if got != -1 {
				t.Errorf("result buffer changed to %d: the function was called", got)
			}
		})
	}

	// The exact count is accepted.
	var got int32
	if _, err = CallFunction(&cif, sym, unsafe.Pointer(&got), ptrs[:3]); err != nil {
		t.Fatal(err)
	}
	if want := int32(1 + 2*2 + 3*3); got != want {
		t.Errorf("int_weighted3(1, 2, 3) = %d, want %d", got, want)
	}
}
