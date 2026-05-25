// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build (linux || darwin || freebsd || windows) && amd64

package ffi

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// TestStructReturn16B_TwoDoubles verifies that {double, double} is returned in XMM0:XMM1.
// This is the NSPoint / NSSize case on macOS Intel — the primary motivation for TASK-045.
// SysV AMD64 ABI: both eightbytes are SSE class → ReturnStXmm0Xmm1.
func TestStructReturn16B_TwoDoubles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows: XMM struct returns not captured by syscall.SyscallN")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "return_struct_2doubles")
	if err != nil {
		t.Fatal(err)
	}

	// {double, double} — both SSE → ReturnStXmm0Xmm1
	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, structType,
		[]*types.TypeDescriptor{types.DoubleTypeDescriptor, types.DoubleTypeDescriptor}); err != nil {
		t.Fatal(err)
	}

	if cif.Flags != types.ReturnStXmm0Xmm1 {
		t.Fatalf("expected cif.Flags = ReturnStXmm0Xmm1 (%d), got %d", types.ReturnStXmm0Xmm1, cif.Flags)
	}

	type PairF64 struct{ A, B float64 }

	a := 1.5
	b := 2.5
	args := []unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b)}
	var result PairF64
	if err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	if result.A != a || result.B != b {
		t.Errorf("return_struct_2doubles(%f, %f) = {%f, %f}, want {%f, %f}",
			a, b, result.A, result.B, a, b)
	}
}

// TestStructReturn16B_IntFloat verifies that {int64, double} returns in RAX:XMM0.
// SysV AMD64 ABI: eightbyte0 INTEGER (RAX), eightbyte1 SSE (XMM0) → ReturnStRaxXmm0.
func TestStructReturn16B_IntFloat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows: XMM struct returns not captured by syscall.SyscallN")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "return_struct_int_float")
	if err != nil {
		t.Fatal(err)
	}

	// {int64, double} — INTEGER + SSE → ReturnStRaxXmm0
	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.DoubleTypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, structType,
		[]*types.TypeDescriptor{types.SInt64TypeDescriptor, types.DoubleTypeDescriptor}); err != nil {
		t.Fatal(err)
	}

	if cif.Flags != types.ReturnStRaxXmm0 {
		t.Fatalf("expected cif.Flags = ReturnStRaxXmm0 (%d), got %d", types.ReturnStRaxXmm0, cif.Flags)
	}

	type MixedIntFloat struct {
		A int64
		B float64
	}

	a := int64(42)
	b := 3.14
	args := []unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b)}
	var result MixedIntFloat
	if err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	if result.A != a || result.B != b {
		t.Errorf("return_struct_int_float(%d, %f) = {%d, %f}, want {%d, %f}",
			a, b, result.A, result.B, a, b)
	}
}

// TestStructReturn16B_FloatInt verifies that {double, int64} returns in XMM0:RAX.
// SysV AMD64 ABI: eightbyte0 SSE (XMM0), eightbyte1 INTEGER (RAX) → ReturnStXmm0Rax.
func TestStructReturn16B_FloatInt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows: XMM struct returns not captured by syscall.SyscallN")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "return_struct_float_int")
	if err != nil {
		t.Fatal(err)
	}

	// {double, int64} — SSE + INTEGER → ReturnStXmm0Rax
	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.SInt64TypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, structType,
		[]*types.TypeDescriptor{types.DoubleTypeDescriptor, types.SInt64TypeDescriptor}); err != nil {
		t.Fatal(err)
	}

	if cif.Flags != types.ReturnStXmm0Rax {
		t.Fatalf("expected cif.Flags = ReturnStXmm0Rax (%d), got %d", types.ReturnStXmm0Rax, cif.Flags)
	}

	type MixedFloatInt struct {
		A float64
		B int64
	}

	a := 2.71828
	b := int64(100)
	args := []unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b)}
	var result MixedFloatInt
	if err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	if result.A != a || result.B != b {
		t.Errorf("return_struct_float_int(%f, %d) = {%f, %d}, want {%f, %d}",
			a, b, result.A, result.B, a, b)
	}
}

// TestStructReturn16B_TwoInts verifies that {int64, int64} returns in RAX:RDX.
// SysV AMD64 ABI: both eightbytes INTEGER → ReturnStRaxRdx.
func TestStructReturn16B_TwoInts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows: 16B struct returns use sret, not RAX:RDX (Win64 ABI)")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "return_struct_2ints")
	if err != nil {
		t.Fatal(err)
	}

	// {int64, int64} — both INTEGER → ReturnStRaxRdx
	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, structType,
		[]*types.TypeDescriptor{types.SInt64TypeDescriptor, types.SInt64TypeDescriptor}); err != nil {
		t.Fatal(err)
	}

	if cif.Flags != types.ReturnStRaxRdx {
		t.Fatalf("expected cif.Flags = ReturnStRaxRdx (%d), got %d", types.ReturnStRaxRdx, cif.Flags)
	}

	type PairI64 struct{ A, B int64 }

	a := int64(1000000)
	b := int64(2000000)
	args := []unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b)}
	var result PairI64
	if err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	if result.A != a || result.B != b {
		t.Errorf("return_struct_2ints(%d, %d) = {%d, %d}, want {%d, %d}",
			a, b, result.A, result.B, a, b)
	}
}
