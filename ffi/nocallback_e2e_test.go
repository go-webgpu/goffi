// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux || darwin || freebsd || windows) && (amd64 || arm64)

package ffi

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

func prepareTestFunc(t *testing.T, name string, ret *types.TypeDescriptor, args ...*types.TypeDescriptor) (*types.CallInterface, unsafe.Pointer) {
	t.Helper()
	requireStructLib(t)
	fn, err := GetSymbol(structTestLib, name)
	if err != nil {
		t.Fatal(err)
	}
	cif := new(types.CallInterface)
	if err := PrepareCallInterface(cif, types.DefaultCall, ret, args); err != nil {
		t.Fatal(err)
	}
	return cif, fn
}

// TestCallFunctionNoCallback passes ten arguments, some on the stack, from
// stack memory and compares the result with CallFunction's.
func TestCallFunctionNoCallback(t *testing.T) {
	args := make([]*types.TypeDescriptor, 10)
	for i := range args {
		args[i] = types.SInt32TypeDescriptor
	}
	cif, fn := prepareTestFunc(t, "stack_i32x10", types.SInt64TypeDescriptor, args...)

	var v [10]int32
	var avalue [10]unsafe.Pointer
	for i := range v {
		v[i] = int32(i*7 - 20)
		avalue[i] = unsafe.Pointer(&v[i])
	}
	var got, want int64
	if _, err := CallFunctionNoCallback(cif, fn, unsafe.Pointer(&got), avalue[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := CallFunction(cif, fn, unsafe.Pointer(&want), avalue[:]); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("CallFunctionNoCallback returned %d, CallFunction %d", got, want)
	}
}

// TestCallFunctionNoCallbackAllocs checks that a call through a bridge
// allocates nothing: the values, avalue and the result stay on the stack.
func TestCallFunctionNoCallbackAllocs(t *testing.T) {
	cif, fn := prepareTestFunc(t, "int_weighted3", types.SInt32TypeDescriptor,
		types.SInt32TypeDescriptor, types.SInt32TypeDescriptor, types.SInt32TypeDescriptor)
	if cif.Bridge == nil {
		t.Skip("no call bridge on this platform")
	}
	var r int32
	allocs := testing.AllocsPerRun(100, func() {
		a, b, c := int32(1), int32(2), int32(3)
		if _, err := CallFunctionNoCallback(cif, fn, unsafe.Pointer(&r),
			[]unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b), unsafe.Pointer(&c)}); err != nil {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("%v allocations per call, want 0", allocs)
	}
	if r != 14 {
		t.Fatalf("int_weighted3(1, 2, 3) = %d, want 14", r)
	}
}

func TestCallFunctionNoCallbackNilArgument(t *testing.T) {
	cif, fn := prepareTestFunc(t, "int_weighted3", types.SInt32TypeDescriptor,
		types.SInt32TypeDescriptor, types.SInt32TypeDescriptor, types.SInt32TypeDescriptor)
	a := int32(1)
	var r int32
	_, err := CallFunctionNoCallback(cif, fn, unsafe.Pointer(&r), []unsafe.Pointer{unsafe.Pointer(&a), nil, unsafe.Pointer(&a)})
	var invalid *InvalidCallInterfaceError
	if !errors.As(err, &invalid) || invalid.Index != 1 {
		t.Fatalf("err = %v, want an InvalidCallInterfaceError for argument 1", err)
	}
	if _, err := CallFunctionNoCallback(cif, fn, unsafe.Pointer(&r), []unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&a), unsafe.Pointer(&a)}); err != nil {
		t.Fatalf("a call after the rejected one failed: %v", err)
	}
}
