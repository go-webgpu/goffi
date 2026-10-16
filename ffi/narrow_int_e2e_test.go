// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux || darwin || freebsd || windows) && (amd64 || arm64)

package ffi

import (
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// Narrow integer arguments: the C functions return the argument widened to 64
// bits, so a value extended differently from what the callee's compiler
// expects shows up as a wrong result (e.g. 253 for int8 -3).

func TestNarrowIntArgs_Registers(t *testing.T) {
	s8, s16 := int8(-3), int16(-1234)
	u8, u16 := uint8(0xFD), uint16(0xFB2E)
	tests := []struct {
		name string
		fn   string
		arg  *types.TypeDescriptor
		val  unsafe.Pointer
		want int64
	}{
		{"int8", "narrow_s8", types.SInt8TypeDescriptor, unsafe.Pointer(&s8), -3},
		{"int16", "narrow_s16", types.SInt16TypeDescriptor, unsafe.Pointer(&s16), -1234},
		{"uint8", "narrow_u8", types.UInt8TypeDescriptor, unsafe.Pointer(&u8), 0xFD},
		{"uint16", "narrow_u16", types.UInt16TypeDescriptor, unsafe.Pointer(&u16), 0xFB2E},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got int64
			if err := callStackFunc(t, tt.fn, types.SInt64TypeDescriptor, unsafe.Pointer(&got),
				[]*types.TypeDescriptor{tt.arg}, []unsafe.Pointer{tt.val}); err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("%s(%d) = %d (%#x), want %d", tt.fn, tt.want, got, uint64(got), tt.want)
			}
		})
	}
}

func TestNarrowIntArgs_Stack(t *testing.T) {
	var regs [8]int64
	argTypes := repeatType(types.SInt64TypeDescriptor, 8)
	args := make([]unsafe.Pointer, 0, 10)
	var want int64
	for i := range regs {
		regs[i] = int64(i + 1)
		args = append(args, unsafe.Pointer(&regs[i]))
		want += regs[i]
	}
	a, b := int8(-3), int16(-1234)
	argTypes = append(argTypes, types.SInt8TypeDescriptor, types.SInt16TypeDescriptor)
	args = append(args, unsafe.Pointer(&a), unsafe.Pointer(&b))
	want += int64(a)*100000 + int64(b)

	var got int64
	if err := callStackFunc(t, "narrow_stack", types.SInt64TypeDescriptor, unsafe.Pointer(&got),
		argTypes, args); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("narrow_stack = %d, want %d", got, want)
	}
}
