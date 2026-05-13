// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build (linux || darwin || freebsd || windows) && amd64

package ffi

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

var structTestLib unsafe.Pointer

func TestMain(m *testing.M) {
	if err := buildStructTestLib(); err != nil {
		// If gcc not available, skip struct e2e tests gracefully.
		// Other tests still run.
		structTestLib = nil
	}
	code := m.Run()
	if structTestLib != nil {
		FreeLibrary(structTestLib)
	}
	os.Exit(code)
}

func buildStructTestLib() error {
	srcPath := filepath.Join("testdata", "structtest.c")
	var soPath string
	switch runtime.GOOS {
	case "darwin":
		soPath = filepath.Join("testdata", "libstructtest.dylib")
	case "windows":
		soPath = filepath.Join("testdata", "structtest.dll")
	default:
		soPath = filepath.Join("testdata", "libstructtest.so")
	}

	cc := os.Getenv("CC")
	if cc == "" {
		cc = "gcc"
	}
	args := []string{"-shared", "-O2", "-o", soPath, srcPath}
	if runtime.GOOS != "windows" {
		args = []string{"-shared", "-fPIC", "-O2", "-o", soPath, srcPath}
	}
	cmd := exec.Command(cc, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	absPath, err := filepath.Abs(soPath)
	if err != nil {
		return err
	}
	lib, err := LoadLibrary(absPath)
	if err != nil {
		return err
	}
	structTestLib = lib
	return nil
}

func requireStructLib(t *testing.T) {
	t.Helper()
	if structTestLib == nil {
		t.Skip("structtest library not available (gcc required)")
	}
}

// TestStructArg8B_IntegerPair tests issue #33: struct {int32, uint32} passed by value.
func TestStructArg8B_IntegerPair(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "take_struct_8")
	if err != nil {
		t.Fatal(err)
	}

	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      8,
		Alignment: 4,
		Members: []*types.TypeDescriptor{
			types.SInt32TypeDescriptor,
			types.UInt32TypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.SInt64TypeDescriptor,
		[]*types.TypeDescriptor{structType}); err != nil {
		t.Fatal(err)
	}

	type Pair struct {
		A int32
		B uint32
	}
	s := Pair{A: 42, B: 19}
	args := []unsafe.Pointer{unsafe.Pointer(&s)}
	var result int64
	if err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	expected := int64(42)*1000 + int64(19)
	if result != expected {
		t.Errorf("take_struct_8({42, 19}) = %d, want %d", result, expected)
	}
}

// TestStructArg8B_FloatPair tests SSE classification: struct {float, float}.
func TestStructArg8B_FloatPair(t *testing.T) {
	requireStructLib(t)
	if runtime.GOOS == "windows" {
		t.Skip("Windows: float struct args/returns not supported via syscall.SyscallN (XMM limitation)")
	}

	sym, err := GetSymbol(structTestLib, "take_struct_2floats")
	if err != nil {
		t.Fatal(err)
	}

	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      8,
		Alignment: 4,
		Members: []*types.TypeDescriptor{
			types.FloatTypeDescriptor,
			types.FloatTypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.FloatTypeDescriptor,
		[]*types.TypeDescriptor{structType}); err != nil {
		t.Fatal(err)
	}

	type PairF32 struct {
		X float32
		Y float32
	}
	s := PairF32{X: 2.5, Y: 3.5}
	args := []unsafe.Pointer{unsafe.Pointer(&s)}
	var result float32
	if err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	if result != 6.0 {
		t.Errorf("take_struct_2floats({2.5, 3.5}) = %f, want 6.0", result)
	}
}

// TestStructArg16B tests two-eightbyte struct: {int64, int64}.
func TestStructArg16B(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "take_struct_16")
	if err != nil {
		t.Fatal(err)
	}

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
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.SInt64TypeDescriptor,
		[]*types.TypeDescriptor{structType}); err != nil {
		t.Fatal(err)
	}

	type PairI64 struct {
		A int64
		B int64
	}
	s := PairI64{A: 1000000, B: 2000000}
	args := []unsafe.Pointer{unsafe.Pointer(&s)}
	var result int64
	if err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	if result != 3000000 {
		t.Errorf("take_struct_16({1000000, 2000000}) = %d, want 3000000", result)
	}
}

// TestStructArg24B_MemoryClass tests > 16B struct passed on stack (MEMORY class).
func TestStructArg24B_MemoryClass(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "take_struct_24")
	if err != nil {
		t.Fatal(err)
	}

	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      24,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.SInt64TypeDescriptor,
		[]*types.TypeDescriptor{structType}); err != nil {
		t.Fatal(err)
	}

	type TripleI64 struct {
		A int64
		B int64
		C int64
	}
	s := TripleI64{A: 100, B: 200, C: 300}
	args := []unsafe.Pointer{unsafe.Pointer(&s)}
	var result int64
	if err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	if result != 600 {
		t.Errorf("take_struct_24({100, 200, 300}) = %d, want 600", result)
	}
}

// TestStructArgWithScalar tests struct + scalar argument (register allocation).
func TestStructArgWithScalar(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "take_struct_and_int")
	if err != nil {
		t.Fatal(err)
	}

	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      8,
		Alignment: 4,
		Members: []*types.TypeDescriptor{
			types.SInt32TypeDescriptor,
			types.UInt32TypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.SInt64TypeDescriptor,
		[]*types.TypeDescriptor{structType, types.SInt64TypeDescriptor}); err != nil {
		t.Fatal(err)
	}

	type Pair struct {
		A int32
		B uint32
	}
	s := Pair{A: 10, B: 20}
	extra := int64(1000)
	args := []unsafe.Pointer{unsafe.Pointer(&s), unsafe.Pointer(&extra)}
	var result int64
	if err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	expected := int64(10) + int64(20) + int64(1000)
	if result != expected {
		t.Errorf("take_struct_and_int({10, 20}, 1000) = %d, want %d", result, expected)
	}
}

func TestCallbackStructArg8B_IntegerPair(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("callback struct args not supported on Windows")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "callback_struct_8")
	if err != nil {
		t.Fatal(err)
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{
			types.SInt32TypeDescriptor,
			types.UInt32TypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatal(err)
	}

	type Pair struct {
		A int32
		B uint32
	}

	var receivedArg Pair
	callback := NewCallback(func(s Pair) {
		receivedArg = s
	})

	expected := Pair{42, 10}
	args := []unsafe.Pointer{
		unsafe.Pointer(&expected.A),
		unsafe.Pointer(&expected.B),
		unsafe.Pointer(&callback),
	}

	if err := CallFunction(&cif, sym, nil, args); err != nil {
		t.Fatal(err)
	}

	if receivedArg != expected {
		t.Errorf("expected %#v, received %#v", expected, receivedArg)
	}
}

func TestCallbackStructArg8B_FloatPair(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("callback struct args not supported on Windows")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "callback_struct_2floats")
	if err != nil {
		t.Fatal(err)
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{
			types.FloatTypeDescriptor,
			types.FloatTypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatal(err)
	}

	type PairF32 struct {
		X float32
		Y float32
	}

	var receivedArg PairF32
	callback := NewCallback(func(s PairF32) {
		receivedArg = s
	})

	expected := PairF32{2.5, 3.5}
	args := []unsafe.Pointer{
		unsafe.Pointer(&expected.X),
		unsafe.Pointer(&expected.Y),
		unsafe.Pointer(&callback),
	}

	if err := CallFunction(&cif, sym, nil, args); err != nil {
		t.Fatal(err)
	}

	if receivedArg != expected {
		t.Errorf("expected %#v, received %#v", expected, receivedArg)
	}
}

func TestCallbackStructArg16B(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("callback struct args not supported on Windows")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "callback_struct_16")
	if err != nil {
		t.Fatal(err)
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatal(err)
	}

	type PairI64 struct {
		A int64
		B int64
	}

	var receivedArg PairI64
	callback := NewCallback(func(s PairI64) {
		receivedArg = s
	})

	expected := PairI64{1000000, 2000000}
	args := []unsafe.Pointer{
		unsafe.Pointer(&expected.A),
		unsafe.Pointer(&expected.B),
		unsafe.Pointer(&callback),
	}

	if err := CallFunction(&cif, sym, nil, args); err != nil {
		t.Fatal(err)
	}

	if receivedArg != expected {
		t.Errorf("expected %#v, received %#v", expected, receivedArg)
	}
}

func TestCallbackStructArg24B_MemoryClass(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("callback struct args not supported on Windows")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "callback_struct_24")
	if err != nil {
		t.Fatal(err)
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatal(err)
	}

	type TripleI64 struct {
		A int64
		B int64
		C int64
	}

	var receivedArg TripleI64
	callback := NewCallback(func(s TripleI64) {
		receivedArg = s
	})

	expected := TripleI64{100, 200, 300}
	args := []unsafe.Pointer{
		unsafe.Pointer(&expected.A),
		unsafe.Pointer(&expected.B),
		unsafe.Pointer(&expected.C),
		unsafe.Pointer(&callback),
	}

	if err := CallFunction(&cif, sym, nil, args); err != nil {
		t.Fatal(err)
	}

	if receivedArg != expected {
		t.Errorf("expected %#v, received %#v", expected, receivedArg)
	}
}

func TestCallbackStructArgWithScalar(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("callback struct args not supported on Windows")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "callback_struct_and_int")
	if err != nil {
		t.Fatal(err)
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{
			types.SInt32TypeDescriptor,
			types.UInt32TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatal(err)
	}

	type Pair struct {
		A int32
		B uint32
	}

	var receivedArg1 Pair
	var receivedArg2 int64
	callback := NewCallback(func(s Pair, extra int64) {
		receivedArg1 = s
		receivedArg2 = extra
	})

	expected := Pair{10, 20}
	extra := int64(1000)
	args := []unsafe.Pointer{
		unsafe.Pointer(&expected.A),
		unsafe.Pointer(&expected.B),
		unsafe.Pointer(&extra),
		unsafe.Pointer(&callback),
	}

	if err := CallFunction(&cif, sym, nil, args); err != nil {
		t.Fatal(err)
	}

	if receivedArg1 != expected || receivedArg2 != extra {
		t.Errorf("expected %#v %d, received %#v %d", expected, extra, receivedArg1, receivedArg2)
	}
}

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
