// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build (linux || darwin || freebsd || windows) && (amd64 || arm64)

package ffi

import (
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// TestVariadic_SumIntegers tests sum_variadic(count int64, ...) with count=3
// and three int64 variadic arguments.  Expected result: 10+20+30 = 60.
//
// This exercises PrepareVariadicCallInterface with nfixedargs=1.
// On Apple ARM64 the variadic arguments must go on the stack; on all other
// platforms the call behaves the same as a non-variadic PrepareCallInterface.
func TestVariadic_SumIntegers(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "sum_variadic")
	if err != nil {
		t.Fatal(err)
	}

	// sum_variadic(int64_t count, ...) — all args are int64_t.
	allArgTypes := []*types.TypeDescriptor{
		types.SInt64TypeDescriptor, // count (fixed)
		types.SInt64TypeDescriptor, // arg 1 (variadic)
		types.SInt64TypeDescriptor, // arg 2 (variadic)
		types.SInt64TypeDescriptor, // arg 3 (variadic)
	}

	var cif types.CallInterface
	if err := PrepareVariadicCallInterface(
		&cif,
		types.DefaultCall,
		1, // nfixedargs: only 'count' is fixed
		types.SInt64TypeDescriptor,
		allArgTypes,
	); err != nil {
		t.Fatal(err)
	}

	count := int64(3)
	a1 := int64(10)
	a2 := int64(20)
	a3 := int64(30)

	avalue := []unsafe.Pointer{
		unsafe.Pointer(&count),
		unsafe.Pointer(&a1),
		unsafe.Pointer(&a2),
		unsafe.Pointer(&a3),
	}

	var result int64
	if err := CallFunction(&cif, sym, unsafe.Pointer(&result), avalue); err != nil {
		t.Fatal(err)
	}

	const want = int64(60)
	if result != want {
		t.Errorf("sum_variadic(3, 10, 20, 30) = %d, want %d", result, want)
	}
}

// TestVariadic_TwoFixed tests variadic_two_fixed(a, b int64, ...) where two
// arguments are fixed and one is variadic.  Expected result: 100+200+300 = 600.
//
// This exercises PrepareVariadicCallInterface with nfixedargs=2.
func TestVariadic_TwoFixed(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "variadic_two_fixed")
	if err != nil {
		t.Fatal(err)
	}

	// variadic_two_fixed(int64_t a, int64_t b, ...) — all args int64_t.
	allArgTypes := []*types.TypeDescriptor{
		types.SInt64TypeDescriptor, // a (fixed)
		types.SInt64TypeDescriptor, // b (fixed)
		types.SInt64TypeDescriptor, // extra (variadic)
	}

	var cif types.CallInterface
	if err := PrepareVariadicCallInterface(
		&cif,
		types.DefaultCall,
		2, // nfixedargs: a and b are fixed
		types.SInt64TypeDescriptor,
		allArgTypes,
	); err != nil {
		t.Fatal(err)
	}

	a := int64(100)
	b := int64(200)
	extra := int64(300)

	avalue := []unsafe.Pointer{
		unsafe.Pointer(&a),
		unsafe.Pointer(&b),
		unsafe.Pointer(&extra),
	}

	var result int64
	if err := CallFunction(&cif, sym, unsafe.Pointer(&result), avalue); err != nil {
		t.Fatal(err)
	}

	const want = int64(600)
	if result != want {
		t.Errorf("variadic_two_fixed(100, 200, 300) = %d, want %d", result, want)
	}
}

// TestVariadic_ErrorValidation verifies that PrepareVariadicCallInterface
// returns an error for invalid nfixedargs values.
func TestVariadic_ErrorValidation(t *testing.T) {
	t.Run("negative nfixedargs", func(t *testing.T) {
		var cif types.CallInterface
		err := PrepareVariadicCallInterface(
			&cif,
			types.DefaultCall,
			-1,
			types.SInt64TypeDescriptor,
			[]*types.TypeDescriptor{types.SInt64TypeDescriptor},
		)
		if err == nil {
			t.Error("expected error for nfixedargs=-1, got nil")
		}
	})

	t.Run("nfixedargs exceeds arg count", func(t *testing.T) {
		var cif types.CallInterface
		err := PrepareVariadicCallInterface(
			&cif,
			types.DefaultCall,
			5, // more than len(argTypes)==1
			types.SInt64TypeDescriptor,
			[]*types.TypeDescriptor{types.SInt64TypeDescriptor},
		)
		if err == nil {
			t.Error("expected error for nfixedargs > len(argTypes), got nil")
		}
	})

	t.Run("nfixedargs equals arg count is allowed", func(t *testing.T) {
		// nfixedargs == len(argTypes) means no variadic args in this particular
		// call, which is valid (caller passes no variadic arguments).
		var cif types.CallInterface
		err := PrepareVariadicCallInterface(
			&cif,
			types.DefaultCall,
			1,
			types.SInt64TypeDescriptor,
			[]*types.TypeDescriptor{types.SInt64TypeDescriptor},
		)
		if err != nil {
			t.Errorf("unexpected error for nfixedargs==len(argTypes): %v", err)
		}
		if cif.FixedArgCount != 1 {
			t.Errorf("cif.FixedArgCount = %d, want 1", cif.FixedArgCount)
		}
	})

	t.Run("zero nfixedargs is allowed", func(t *testing.T) {
		// nfixedargs == 0 is legal — all args are variadic.
		var cif types.CallInterface
		err := PrepareVariadicCallInterface(
			&cif,
			types.DefaultCall,
			0,
			types.SInt64TypeDescriptor,
			[]*types.TypeDescriptor{types.SInt64TypeDescriptor},
		)
		if err != nil {
			t.Errorf("unexpected error for nfixedargs=0: %v", err)
		}
		if cif.FixedArgCount != 0 {
			t.Errorf("cif.FixedArgCount = %d, want 0", cif.FixedArgCount)
		}
	})
}
