// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux || darwin || freebsd || windows) && (amd64 || arm64)

package ffi

import (
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// Arguments passed on the stack (#96): the C functions in testdata/structtest.c
// weight every argument by its position, so one placed at the wrong stack
// offset changes the result.

// callStackFunc calls name from the C test library and returns CallFunction's
// error instead of failing, so a test can accept an explicit "too many
// arguments" error but not a silently wrong result.
func callStackFunc(t *testing.T, name string, ret *types.TypeDescriptor, rvalue unsafe.Pointer,
	argTypes []*types.TypeDescriptor, args []unsafe.Pointer) error {
	t.Helper()
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, name)
	if err != nil {
		t.Fatal(err)
	}
	var cif types.CallInterface
	if err = PrepareCallInterface(&cif, types.DefaultCall, ret, argTypes); err != nil {
		return err
	}
	_, err = CallFunction(&cif, sym, rvalue, args)
	return err
}

func repeatType(td *types.TypeDescriptor, n int) []*types.TypeDescriptor {
	out := make([]*types.TypeDescriptor, n)
	for i := range out {
		out[i] = td
	}
	return out
}

func TestStackArgs_Int32x10(t *testing.T) {
	var in [10]int32
	args := make([]unsafe.Pointer, len(in))
	var want int64
	for i := range in {
		in[i] = int32(i+1) * 1000
		args[i] = unsafe.Pointer(&in[i])
		want += int64(i+1) * int64(in[i])
	}
	var got int64
	if err := callStackFunc(t, "stack_i32x10", types.SInt64TypeDescriptor, unsafe.Pointer(&got),
		repeatType(types.SInt32TypeDescriptor, 10), args); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("stack_i32x10 = %d, want %d", got, want)
	}
}

func TestStackArgs_Float32x10(t *testing.T) {
	var in [10]float32
	args := make([]unsafe.Pointer, len(in))
	var want float32
	for i := range in {
		in[i] = float32(i + 1)
		args[i] = unsafe.Pointer(&in[i])
		want += float32(i+1) * in[i]
	}
	var got float32
	if err := callStackFunc(t, "stack_f32x10", types.FloatTypeDescriptor, unsafe.Pointer(&got),
		repeatType(types.FloatTypeDescriptor, 10), args); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("stack_f32x10 = %v, want %v", got, want)
	}
}

func TestStackArgs_NarrowMix(t *testing.T) {
	var regs [8]int64
	argTypes := repeatType(types.SInt64TypeDescriptor, 8)
	args := make([]unsafe.Pointer, 0, 12)
	var want int64
	for i := range regs {
		regs[i] = int64(i + 1)
		args = append(args, unsafe.Pointer(&regs[i]))
		want += regs[i]
	}
	a, b, c, d := int8(-3), int16(-1234), int32(56789), int8(7)
	argTypes = append(argTypes, types.SInt8TypeDescriptor, types.SInt16TypeDescriptor,
		types.SInt32TypeDescriptor, types.SInt8TypeDescriptor)
	args = append(args, unsafe.Pointer(&a), unsafe.Pointer(&b), unsafe.Pointer(&c), unsafe.Pointer(&d))
	want += int64(a)*1000000000 + int64(b)*100000 + int64(c)*10 + int64(d)

	var got int64
	if err := callStackFunc(t, "stack_narrow_mix", types.SInt64TypeDescriptor, unsafe.Pointer(&got),
		argTypes, args); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("stack_narrow_mix = %d, want %d", got, want)
	}
}

func TestStackArgs_HFAAfterEightFloats(t *testing.T) {
	var regs [8]float32
	argTypes := repeatType(types.FloatTypeDescriptor, 8)
	args := make([]unsafe.Pointer, 0, 9)
	var want float32
	for i := range regs {
		regs[i] = float32(i + 1)
		args = append(args, unsafe.Pointer(&regs[i]))
		want += regs[i]
	}
	type pairF32 struct{ X, Y float32 }
	s := pairF32{X: 3, Y: 5}
	argTypes = append(argTypes, &types.TypeDescriptor{
		Kind: types.StructType, Size: 8, Alignment: 4,
		Members: []*types.TypeDescriptor{types.FloatTypeDescriptor, types.FloatTypeDescriptor},
	})
	args = append(args, unsafe.Pointer(&s))
	want += s.X*10 + s.Y*100

	var got float32
	if err := callStackFunc(t, "stack_hfa_after_8f", types.FloatTypeDescriptor, unsafe.Pointer(&got),
		argTypes, args); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("stack_hfa_after_8f = %v, want %v", got, want)
	}
}

func TestStackArgs_Struct16AfterSevenInts(t *testing.T) {
	var regs [7]int64
	argTypes := repeatType(types.SInt64TypeDescriptor, 7)
	args := make([]unsafe.Pointer, 0, 8)
	var want int64
	for i := range regs {
		regs[i] = int64(i + 1)
		args = append(args, unsafe.Pointer(&regs[i]))
		want += regs[i]
	}
	type pairI64 struct{ A, B int64 }
	s := pairI64{A: 42, B: 7}
	argTypes = append(argTypes, &types.TypeDescriptor{
		Kind: types.StructType, Size: 16, Alignment: 8,
		Members: []*types.TypeDescriptor{types.SInt64TypeDescriptor, types.SInt64TypeDescriptor},
	})
	args = append(args, unsafe.Pointer(&s))
	want += s.A*1000 + s.B

	var got int64
	if err := callStackFunc(t, "stack_s16_after_7i", types.SInt64TypeDescriptor, unsafe.Pointer(&got),
		argTypes, args); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("stack_s16_after_7i = %d, want %d", got, want)
	}
}

// More arguments than the call path supports must be an error, never a call
// with some arguments dropped.
func TestStackArgs_TooManyIsAnError(t *testing.T) {
	var in [16]int64
	args := make([]unsafe.Pointer, len(in))
	var want int64
	for i := range in {
		in[i] = int64(i + 1)
		args[i] = unsafe.Pointer(&in[i])
		want += in[i]
	}
	want += 15 * in[15] // the C function weights the last argument by 16

	var got int64
	err := callStackFunc(t, "stack_i64x16", types.SInt64TypeDescriptor, unsafe.Pointer(&got),
		repeatType(types.SInt64TypeDescriptor, 16), args)
	if err != nil {
		t.Logf("16 arguments rejected: %v", err)
		return
	}
	if got != want {
		t.Errorf("stack_i64x16 returned %d without an error, want %d or an error", got, want)
	}
}

// Sixteen doubles put eight on the stack: either the call path supports that,
// or the call fails. It must not drop the arguments that do not fit.
func TestStackArgs_SixteenDoubles(t *testing.T) {
	var in [16]float64
	args := make([]unsafe.Pointer, len(in))
	var want float64
	for i := range in {
		in[i] = float64(i + 1)
		args[i] = unsafe.Pointer(&in[i])
		want += in[i]
	}
	want += 15 * in[15] // the C function weights the last argument by 16

	var got float64
	err := callStackFunc(t, "stack_f64x16", types.DoubleTypeDescriptor, unsafe.Pointer(&got),
		repeatType(types.DoubleTypeDescriptor, 16), args)
	if err != nil {
		t.Logf("16 doubles rejected: %v", err)
		return
	}
	if got != want {
		t.Errorf("stack_f64x16 returned %v without an error, want %v or an error", got, want)
	}
}
