// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux || darwin || freebsd || windows) && (amd64 || arm64)

package ffi

import (
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// types.IntTypeDescriptor describes C int (#99): a signed 32-bit integer on
// every supported target, passed and returned like SInt32TypeDescriptor.

// Ten int arguments put some on the stack on every target (6 GP registers on
// SysV, 4 on Win64, 8 on arm64). Alternating signs catch a missing sign
// extension as well as an argument passed as a pointer.
func TestIntType_TenArgs(t *testing.T) {
	var in [10]int32
	args := make([]unsafe.Pointer, len(in))
	var want int64
	for i := range in {
		in[i] = int32(i+1) * 1000
		if i%2 == 1 {
			in[i] = -in[i]
		}
		args[i] = unsafe.Pointer(&in[i])
		want += int64(i+1) * int64(in[i])
	}
	var got int64
	if err := callStackFunc(t, "stack_i32x10", types.SInt64TypeDescriptor, unsafe.Pointer(&got),
		repeatType(types.IntTypeDescriptor, 10), args); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("stack_i32x10 with IntTypeDescriptor arguments = %d, want %d", got, want)
	}
}

// The argument and result directions are checked separately, so a failure
// names the broken one.
func TestIntType_ArgsAndResult(t *testing.T) {
	a, b, c := int32(-7), int32(100), int32(-1000)
	const want = int32(-7 + 2*100 + 3*-1000)

	tests := []struct {
		name string
		arg  *types.TypeDescriptor
		ret  *types.TypeDescriptor
	}{
		{"int args, int result", types.IntTypeDescriptor, types.IntTypeDescriptor},
		{"int args, int32 result", types.IntTypeDescriptor, types.SInt32TypeDescriptor},
		{"int32 args, int result", types.SInt32TypeDescriptor, types.IntTypeDescriptor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := int32(0x5a5a5a5a)
			err := callStackFunc(t, "int_weighted3", tt.ret, unsafe.Pointer(&got),
				repeatType(tt.arg, 3), []unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b), unsafe.Pointer(&c)})
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("int_weighted3 = %d, want %d", got, want)
			}
		})
	}
}
