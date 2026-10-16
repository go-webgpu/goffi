// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build ((linux && !android) || darwin || freebsd) && (amd64 || arm64)

package ffi

import (
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// Callbacks with arguments on the stack (#96): C calls the Go callback with
// 1..10, two or more of which are passed on the stack.

func TestCallbackE2E_StackInt32x10(t *testing.T) {
	var got [10]int32
	cb := NewCallback(func(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9 int32) int64 {
		got = [10]int32{a0, a1, a2, a3, a4, a5, a6, a7, a8, a9}
		return int64(a8)*100 + int64(a9)
	})

	var result int64
	callCFunc(t, "cb_call_stack_i32x10", types.SInt64TypeDescriptor, unsafe.Pointer(&result),
		[]*types.TypeDescriptor{types.PointerTypeDescriptor}, unsafe.Pointer(&cb))

	if want := [10]int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}; got != want {
		t.Errorf("callback received %v, want %v", got, want)
	}
	if result != 910 {
		t.Errorf("C got %d, want 910", result)
	}
}

func TestCallbackE2E_StackFloat32x10(t *testing.T) {
	var got [10]float32
	cb := NewCallback(func(a0, a1, a2, a3, a4, a5, a6, a7, a8, a9 float32) float32 {
		got = [10]float32{a0, a1, a2, a3, a4, a5, a6, a7, a8, a9}
		return a8*100 + a9
	})

	var result float32
	callCFunc(t, "cb_call_stack_f32x10", types.FloatTypeDescriptor, unsafe.Pointer(&result),
		[]*types.TypeDescriptor{types.PointerTypeDescriptor}, unsafe.Pointer(&cb))

	if want := [10]float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}; got != want {
		t.Errorf("callback received %v, want %v", got, want)
	}
	if result != 910 {
		t.Errorf("C got %v, want 910", result)
	}
}
