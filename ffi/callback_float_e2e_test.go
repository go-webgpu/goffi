// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build ((linux && !android) || darwin || freebsd) && (amd64 || arm64)

package ffi

import (
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// These tests let C code from testdata/structtest.c call Go callbacks with
// float and double arguments and results (#93). The C compiler decides where
// each value goes, so a callback that decodes registers or stack slots wrongly,
// or a dispatcher that does not set the floating-point result register, fails
// here even if the hand-built frames in callback_test.go agree with it.

// callCFunc calls the C function name from the test library through goffi.
func callCFunc(t *testing.T, name string, ret *types.TypeDescriptor, rvalue unsafe.Pointer,
	argTypes []*types.TypeDescriptor, args ...unsafe.Pointer) {
	t.Helper()
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, name)
	if err != nil {
		t.Fatal(err)
	}
	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, ret, argTypes); err != nil {
		t.Fatal(err)
	}
	if _, err := CallFunction(&cif, sym, rvalue, args); err != nil {
		t.Fatal(err)
	}
}

func TestCallbackE2E_FloatArgs(t *testing.T) {
	var gotA, gotB float32
	cb := NewCallback(func(a, b float32) int32 {
		gotA, gotB = a, b
		return int32(a*2 + b*4)
	})

	a, b := float32(2.5), float32(4)
	var result int32
	callCFunc(t, "cb_call_float_args", types.SInt32TypeDescriptor, unsafe.Pointer(&result),
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.FloatTypeDescriptor, types.FloatTypeDescriptor},
		unsafe.Pointer(&cb), unsafe.Pointer(&a), unsafe.Pointer(&b))

	if gotA != a || gotB != b {
		t.Errorf("callback received (%v, %v), want (%v, %v)", gotA, gotB, a, b)
	}
	if result != 21 {
		t.Errorf("C got %d from the callback, want 21", result)
	}
}

func TestCallbackE2E_DoubleArgs(t *testing.T) {
	var gotA, gotB float64
	cb := NewCallback(func(a, b float64) int32 {
		gotA, gotB = a, b
		return int32(a*2 + b*4)
	})

	a, b := 2.5, 4.0
	var result int32
	callCFunc(t, "cb_call_double_args", types.SInt32TypeDescriptor, unsafe.Pointer(&result),
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.DoubleTypeDescriptor, types.DoubleTypeDescriptor},
		unsafe.Pointer(&cb), unsafe.Pointer(&a), unsafe.Pointer(&b))

	if gotA != a || gotB != b {
		t.Errorf("callback received (%v, %v), want (%v, %v)", gotA, gotB, a, b)
	}
	if result != 21 {
		t.Errorf("C got %d from the callback, want 21", result)
	}
}

func TestCallbackE2E_FloatReturn(t *testing.T) {
	cb := NewCallback(func(a int64) float32 { return float32(a) + 0.5 })

	a := int64(41)
	var result float32
	callCFunc(t, "cb_call_float_ret", types.FloatTypeDescriptor, unsafe.Pointer(&result),
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.SInt64TypeDescriptor},
		unsafe.Pointer(&cb), unsafe.Pointer(&a))

	if result != 41.5 {
		t.Errorf("C got %v from the callback, want 41.5", result)
	}
}

func TestCallbackE2E_DoubleReturn(t *testing.T) {
	cb := NewCallback(func(a int64) float64 { return float64(a) * 1.25 })

	a := int64(3)
	var result float64
	callCFunc(t, "cb_call_double_ret", types.DoubleTypeDescriptor, unsafe.Pointer(&result),
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.SInt64TypeDescriptor},
		unsafe.Pointer(&cb), unsafe.Pointer(&a))

	if result != 3.75 {
		t.Errorf("C got %v from the callback, want 3.75", result)
	}
}

func TestCallbackE2E_MixedArgs(t *testing.T) {
	type received struct {
		a int32
		b float32
		c int64
		d float64
		e float32
	}
	var got received
	cb := NewCallback(func(a int32, b float32, c int64, d float64, e float32) float64 {
		got = received{a, b, c, d, e}
		return float64(a) + float64(b) + d + float64(e)
	})

	want := received{a: -7, b: 1.5, c: 1 << 40, d: 0.25, e: -2.75}
	var result float64
	callCFunc(t, "cb_call_mixed", types.DoubleTypeDescriptor, unsafe.Pointer(&result),
		[]*types.TypeDescriptor{
			types.PointerTypeDescriptor, types.SInt32TypeDescriptor, types.FloatTypeDescriptor,
			types.SInt64TypeDescriptor, types.DoubleTypeDescriptor, types.FloatTypeDescriptor,
		},
		unsafe.Pointer(&cb), unsafe.Pointer(&want.a), unsafe.Pointer(&want.b),
		unsafe.Pointer(&want.c), unsafe.Pointer(&want.d), unsafe.Pointer(&want.e))

	if got != want {
		t.Errorf("callback received %+v, want %+v", got, want)
	}
	if result != -8 {
		t.Errorf("C got %v from the callback, want -8", result)
	}
}

// The ninth float is passed on the stack; the result goes back through the
// float result register and C adds 0.25 to it.
func TestCallbackE2E_FloatStackArg(t *testing.T) {
	var got [9]float32
	cb := NewCallback(func(a, b, c, d, e, f, g, h, i float32) float32 {
		got = [9]float32{a, b, c, d, e, f, g, h, i}
		var sum float32
		for k, v := range got {
			sum += float32(k+1) * v
		}
		return sum
	})

	var result float32
	callCFunc(t, "cb_call_float9", types.FloatTypeDescriptor, unsafe.Pointer(&result),
		[]*types.TypeDescriptor{types.PointerTypeDescriptor}, unsafe.Pointer(&cb))

	want := [9]float32{1, 2, 3, 4, 5, 6, 7, 8, 9.5}
	if got != want {
		t.Errorf("callback received %v, want %v", got, want)
	}
	// 1·1 + 2·2 + … + 8·8 + 9·9.5 = 289.5, plus the 0.25 added by C.
	if result != 289.75 {
		t.Errorf("C got %v, want 289.75", result)
	}
}

// Eight doubles fill the registers; a float and a double follow on the stack.
func TestCallbackE2E_StackFloatAndDouble(t *testing.T) {
	var gotF float32
	var gotD float64
	cb := NewCallback(func(a1, a2, a3, a4, a5, a6, a7, a8 float64, f float32, d float64) float64 {
		gotF, gotD = f, d
		return a1 + a2 + a3 + a4 + a5 + a6 + a7 + a8 + float64(f)*100 + d*1000
	})

	var result float64
	callCFunc(t, "cb_call_stack_float_double", types.DoubleTypeDescriptor, unsafe.Pointer(&result),
		[]*types.TypeDescriptor{types.PointerTypeDescriptor}, unsafe.Pointer(&cb))

	if gotF != 9.5 || gotD != 10.25 {
		t.Errorf("callback received stack args (%v, %v), want (9.5, 10.25)", gotF, gotD)
	}
	// 1+…+8 = 36, 9.5·100 = 950, 10.25·1000 = 10250.
	if result != 11236 {
		t.Errorf("C got %v, want 11236", result)
	}
}

// An integer result must still come back in the integer register now that
// the dispatcher also writes the floating-point result register.
func TestCallbackE2E_IntReturnWithDoubleArg(t *testing.T) {
	cb := NewCallback(func(d float64) int64 { return int64(d*4) - (1 << 40) })

	d := 2.5
	var result int64
	callCFunc(t, "cb_call_int_ret", types.SInt64TypeDescriptor, unsafe.Pointer(&result),
		[]*types.TypeDescriptor{types.PointerTypeDescriptor, types.DoubleTypeDescriptor},
		unsafe.Pointer(&cb), unsafe.Pointer(&d))

	if want := int64(10) - (1 << 40); result != want {
		t.Errorf("C got %d, want %d", result, want)
	}
}
