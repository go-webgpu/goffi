// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux || darwin || freebsd || windows) && (amd64 || arm64)

package ffi

import (
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
