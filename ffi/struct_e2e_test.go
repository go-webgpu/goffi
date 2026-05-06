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
