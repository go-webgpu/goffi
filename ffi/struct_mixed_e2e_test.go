// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux || darwin || freebsd) && (amd64 || arm64)

package ffi

import (
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// Structs that mix integer and floating-point members, passed and returned by
// value on every Unix target. SysV amd64 classifies each eightbyte on its own
// (one may go in RAX, the other in XMM0); AAPCS64 passes and returns a
// composite that is not an HFA in X registers, floating-point members included.

func mixedStructType(first, second *types.TypeDescriptor) *types.TypeDescriptor {
	return &types.TypeDescriptor{
		Kind: types.StructType, Size: 16, Alignment: 8,
		Members: []*types.TypeDescriptor{first, second},
	}
}

func TestStructMixed_ArgIntFloat(t *testing.T) {
	type intFloat struct {
		A int64
		B float64
	}
	s := intFloat{A: 42, B: 2.5}
	var got int64
	if err := callStackFunc(t, "take_mixed_int_float", types.SInt64TypeDescriptor, unsafe.Pointer(&got),
		[]*types.TypeDescriptor{mixedStructType(types.SInt64TypeDescriptor, types.DoubleTypeDescriptor)},
		[]unsafe.Pointer{unsafe.Pointer(&s)}); err != nil {
		t.Fatal(err)
	}
	if want := int64(42*1000 + 25); got != want {
		t.Errorf("take_mixed_int_float({42, 2.5}) = %d, want %d", got, want)
	}
}

func TestStructMixed_ArgFloatInt(t *testing.T) {
	type floatInt struct {
		A float64
		B int64
	}
	s := floatInt{A: 2.5, B: 42}
	var got int64
	if err := callStackFunc(t, "take_mixed_float_int", types.SInt64TypeDescriptor, unsafe.Pointer(&got),
		[]*types.TypeDescriptor{mixedStructType(types.DoubleTypeDescriptor, types.SInt64TypeDescriptor)},
		[]unsafe.Pointer{unsafe.Pointer(&s)}); err != nil {
		t.Fatal(err)
	}
	if want := int64(25*1000 + 42); got != want {
		t.Errorf("take_mixed_float_int({2.5, 42}) = %d, want %d", got, want)
	}
}

func TestStructMixed_ReturnIntFloat(t *testing.T) {
	type intFloat struct {
		A int64
		B float64
	}
	a, b := int64(42), 2.5
	var got intFloat
	if err := callStackFunc(t, "return_struct_int_float",
		mixedStructType(types.SInt64TypeDescriptor, types.DoubleTypeDescriptor), unsafe.Pointer(&got),
		[]*types.TypeDescriptor{types.SInt64TypeDescriptor, types.DoubleTypeDescriptor},
		[]unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b)}); err != nil {
		t.Fatal(err)
	}
	if got != (intFloat{A: a, B: b}) {
		t.Errorf("return_struct_int_float(42, 2.5) = %+v", got)
	}
}

func TestStructMixed_ReturnFloatInt(t *testing.T) {
	type floatInt struct {
		A float64
		B int64
	}
	a, b := 2.5, int64(42)
	var got floatInt
	if err := callStackFunc(t, "return_struct_float_int",
		mixedStructType(types.DoubleTypeDescriptor, types.SInt64TypeDescriptor), unsafe.Pointer(&got),
		[]*types.TypeDescriptor{types.DoubleTypeDescriptor, types.SInt64TypeDescriptor},
		[]unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b)}); err != nil {
		t.Fatal(err)
	}
	if got != (floatInt{A: a, B: b}) {
		t.Errorf("return_struct_float_int(2.5, 42) = %+v", got)
	}
}
