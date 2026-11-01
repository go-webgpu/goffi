// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux || darwin || freebsd || windows) && (amd64 || arm64)

package ffi

import (
	"errors"
	"runtime"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// Stack-argument layout (TASK-098): by-value structs that need more stack than
// a fixed area holds, and arguments that must go to the stack whole because
// the registers left cannot take all of their eightbytes (psABI §3.2.3, libffi
// src/x86/ffi64.c). The C functions in testdata/structtest.c weight every
// field and argument, so a misplaced word changes the result.

var (
	u64Type = types.UInt64TypeDescriptor
	u32Type = types.UInt32TypeDescriptor

	// svType is webgpu.h's WGPUStringView: {data, length}, 16 bytes.
	svType = &types.TypeDescriptor{
		Kind: types.StructType, Size: 16, Alignment: 8,
		Members: []*types.TypeDescriptor{u64Type, u64Type},
	}
	// info96Type is the shape of webgpu.h's WGPUAdapterInfo.
	info96Type = &types.TypeDescriptor{
		Kind: types.StructType, Size: 96, Alignment: 8,
		Members: []*types.TypeDescriptor{
			u64Type, svType, svType, svType, svType,
			u32Type, u32Type, u32Type, u32Type, u32Type, u32Type,
		},
	}
	words8Type = &types.TypeDescriptor{
		Kind: types.StructType, Size: 64, Alignment: 8,
		Members: []*types.TypeDescriptor{u64Type, u64Type, u64Type, u64Type, u64Type, u64Type, u64Type, u64Type},
	}
	cbinfo40Type = &types.TypeDescriptor{
		Kind: types.StructType, Size: 40, Alignment: 8,
		Members: []*types.TypeDescriptor{u64Type, u32Type, u64Type, u64Type, u64Type},
	}
	pairI64Type = &types.TypeDescriptor{
		Kind: types.StructType, Size: 16, Alignment: 8,
		Members: []*types.TypeDescriptor{types.SInt64TypeDescriptor, types.SInt64TypeDescriptor},
	}
	pairF64Type = &types.TypeDescriptor{
		Kind: types.StructType, Size: 16, Alignment: 8,
		Members: []*types.TypeDescriptor{types.DoubleTypeDescriptor, types.DoubleTypeDescriptor},
	}
	mixedIntFloatType = &types.TypeDescriptor{
		Kind: types.StructType, Size: 16, Alignment: 8,
		Members: []*types.TypeDescriptor{types.SInt64TypeDescriptor, types.DoubleTypeDescriptor},
	}
)

type stringView struct{ Data, Length uint64 }

type adapterInfo96 struct {
	Next                                      uint64
	Vendor, Architecture, Device, Description stringView
	Backend, AdapterType, VendorID, DeviceID  uint32
	SubgroupMin, SubgroupMax                  uint32
}

// sum mirrors info96_sum in structtest.c.
func (s *adapterInfo96) sum() uint64 {
	return s.Next + 2*s.Vendor.Data + 3*s.Vendor.Length + 4*s.Architecture.Data +
		5*s.Architecture.Length + 6*s.Device.Data + 7*s.Device.Length +
		8*s.Description.Data + 9*s.Description.Length + 10*uint64(s.Backend) +
		11*uint64(s.AdapterType) + 12*uint64(s.VendorID) + 13*uint64(s.DeviceID) +
		14*uint64(s.SubgroupMin) + 15*uint64(s.SubgroupMax)
}

// newAdapterInfo96 fills every field with a distinct value derived from base.
func newAdapterInfo96(base uint64) adapterInfo96 {
	return adapterInfo96{
		Next:         base + 1,
		Vendor:       stringView{base + 2, base + 3},
		Architecture: stringView{base + 4, base + 5},
		Device:       stringView{base + 6, base + 7},
		Description:  stringView{base + 8, base + 9},
		Backend:      uint32(base + 10), AdapterType: uint32(base + 11),
		VendorID: uint32(base + 12), DeviceID: uint32(base + 13),
		SubgroupMin: uint32(base + 14), SubgroupMax: uint32(base + 15),
	}
}

func TestStackLayout_Struct96(t *testing.T) {
	if unsafe.Sizeof(adapterInfo96{}) != 96 {
		t.Fatalf("adapterInfo96 is %d bytes, want 96", unsafe.Sizeof(adapterInfo96{}))
	}
	s := newAdapterInfo96(100)
	var got uint64
	if err := callStackFunc(t, "take_struct_96", u64Type, unsafe.Pointer(&got),
		[]*types.TypeDescriptor{info96Type}, []unsafe.Pointer{unsafe.Pointer(&s)}); err != nil {
		t.Fatal(err)
	}
	if want := s.sum(); got != want {
		t.Errorf("take_struct_96 = %d, want %d", got, want)
	}
}

func TestStackLayout_Struct64(t *testing.T) {
	var s [8]uint64
	var want uint64
	for i := range s {
		s[i] = uint64(i+1) * 1000
		want += uint64(i+1) * s[i]
	}
	var got uint64
	if err := callStackFunc(t, "take_struct_64", u64Type, unsafe.Pointer(&got),
		[]*types.TypeDescriptor{words8Type}, []unsafe.Pointer{unsafe.Pointer(&s)}); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("take_struct_64 = %d, want %d", got, want)
	}
}

// The shape of wgpuBufferMapAsync: four integer arguments, then a 40-byte
// struct by value.
func TestStackLayout_MapAsyncShape(t *testing.T) {
	type cbinfo40 struct {
		Next     uint64
		Mode     uint32
		Callback uint64
		UD1, UD2 uint64
	}
	buffer, mode, offset, size := uint64(3), uint32(5), uint64(7), uint64(11)
	info := cbinfo40{Next: 13, Mode: 17, Callback: 19, UD1: 23, UD2: 29}
	want := buffer + 2*uint64(mode) + 3*offset + 5*size + 7*info.Next + 11*uint64(info.Mode) +
		13*info.Callback + 17*info.UD1 + 19*info.UD2
	var got uint64
	if err := callStackFunc(t, "map_async_like", u64Type, unsafe.Pointer(&got),
		[]*types.TypeDescriptor{u64Type, u32Type, u64Type, u64Type, cbinfo40Type},
		[]unsafe.Pointer{unsafe.Pointer(&buffer), unsafe.Pointer(&mode), unsafe.Pointer(&offset),
			unsafe.Pointer(&size), unsafe.Pointer(&info)}); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("map_async_like = %d, want %d", got, want)
	}
}

// Two 96-byte structs and two scalars: 24 stack words on SysV amd64.
func TestStackLayout_TwoStruct96(t *testing.T) {
	a, b := int64(3), int64(5)
	x, y := newAdapterInfo96(100), newAdapterInfo96(200)
	want := uint64(a) + 1000*x.sum() + 7*uint64(b) + 100000*y.sum()
	var got uint64
	if err := callStackFunc(t, "two_info96", u64Type, unsafe.Pointer(&got),
		[]*types.TypeDescriptor{types.SInt64TypeDescriptor, info96Type, types.SInt64TypeDescriptor, info96Type},
		[]unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&x), unsafe.Pointer(&b), unsafe.Pointer(&y)}); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("two_info96 = %d, want %d", got, want)
	}
}

// Five int64 leave one GPR: the 16-byte struct goes on the stack whole and the
// int64 after it takes R9 (System V). On arm64 and Win64 the same signature
// has registers to spare or passes the struct by reference.
func TestStackLayout_SplitS16After5I(t *testing.T) {
	var regs [5]int64
	args := make([]unsafe.Pointer, 0, 7)
	var want int64
	for i := range regs {
		regs[i] = int64(i + 1)
		args = append(args, unsafe.Pointer(&regs[i]))
		want += int64(i+1) * regs[i]
	}
	type pairI64 struct{ A, B int64 }
	s := pairI64{A: 42, B: 7}
	last := int64(9)
	args = append(args, unsafe.Pointer(&s), unsafe.Pointer(&last))
	want += 1000*s.A + 100*s.B + 100000*last

	argTypes := append(repeatType(types.SInt64TypeDescriptor, 5), pairI64Type, types.SInt64TypeDescriptor)
	var got int64
	if err := callStackFunc(t, "split_s16_after_5i", types.SInt64TypeDescriptor, unsafe.Pointer(&got),
		argTypes, args); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("split_s16_after_5i = %d, want %d", got, want)
	}
}

// Seven doubles leave XMM7: {double, double} goes on the stack whole and the
// double after it takes XMM7 (System V).
func TestStackLayout_SplitD2After7D(t *testing.T) {
	var regs [7]float64
	args := make([]unsafe.Pointer, 0, 9)
	var want float64
	for i := range regs {
		regs[i] = float64(i + 1)
		args = append(args, unsafe.Pointer(&regs[i]))
		want += float64(i+1) * regs[i]
	}
	type pairF64 struct{ A, B float64 }
	s := pairF64{A: 42, B: 7}
	last := 9.0
	args = append(args, unsafe.Pointer(&s), unsafe.Pointer(&last))
	want += 1000*s.A + 100*s.B + 100000*last

	argTypes := append(repeatType(types.DoubleTypeDescriptor, 7), pairF64Type, types.DoubleTypeDescriptor)
	var got float64
	if err := callStackFunc(t, "split_d2_after_7d", types.DoubleTypeDescriptor, unsafe.Pointer(&got),
		argTypes, args); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("split_d2_after_7d = %v, want %v", got, want)
	}
}

// Six int64 use every GPR: {int64, double} needs a GPR and an SSE register, so
// it goes on the stack whole and the double after it takes XMM0 (System V).
func TestStackLayout_SplitMixedAfter6I(t *testing.T) {
	var regs [6]int64
	args := make([]unsafe.Pointer, 0, 8)
	var wantInts int64
	for i := range regs {
		regs[i] = int64(i + 1)
		args = append(args, unsafe.Pointer(&regs[i]))
		wantInts += int64(i+1) * regs[i]
	}
	type mixed struct {
		A int64
		B float64
	}
	s := mixed{A: 42, B: 7}
	last := 9.0
	args = append(args, unsafe.Pointer(&s), unsafe.Pointer(&last))
	want := float64(wantInts) + 1000*float64(s.A) + 100*s.B + 100000*last

	argTypes := append(repeatType(types.SInt64TypeDescriptor, 6), mixedIntFloatType, types.DoubleTypeDescriptor)
	var got float64
	if err := callStackFunc(t, "split_mixed_after_6i", types.DoubleTypeDescriptor, unsafe.Pointer(&got),
		argTypes, args); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("split_mixed_after_6i = %v, want %v", got, want)
	}
}

// A CallInterface filled in by hand skips PrepareCallInterface, so the call
// itself must place every argument or return an error, never panic.
func TestStackLayout_HandBuiltCIF(t *testing.T) {
	requireStructLib(t)

	t.Run("sixteen int64", func(t *testing.T) {
		sym, err := GetSymbol(structTestLib, "stack_i64x16")
		if err != nil {
			t.Fatal(err)
		}
		var in [16]int64
		args := make([]unsafe.Pointer, len(in))
		var want int64
		for i := range in {
			in[i] = int64(i + 1)
			args[i] = unsafe.Pointer(&in[i])
			want += in[i]
		}
		want += 15 * in[15]
		cif := types.CallInterface{ReturnType: types.SInt64TypeDescriptor,
			ArgTypes: repeatType(types.SInt64TypeDescriptor, 16)}
		var got int64
		if _, err := CallFunction(&cif, sym, unsafe.Pointer(&got), args); err != nil {
			t.Logf("rejected: %v", err)
			return
		}
		if got != want {
			t.Errorf("stack_i64x16 = %d without an error, want %d or an error", got, want)
		}
	})

	t.Run("96-byte struct", func(t *testing.T) {
		sym, err := GetSymbol(structTestLib, "take_struct_96")
		if err != nil {
			t.Fatal(err)
		}
		s := newAdapterInfo96(300)
		cif := types.CallInterface{ReturnType: u64Type, ArgTypes: []*types.TypeDescriptor{info96Type}}
		var got uint64
		if _, err := CallFunction(&cif, sym, unsafe.Pointer(&got), []unsafe.Pointer{unsafe.Pointer(&s)}); err != nil {
			t.Fatal(err)
		}
		if want := s.sum(); got != want {
			t.Errorf("take_struct_96 = %d, want %d", got, want)
		}
	})
}

// Arguments past every platform's limit are an error at prepare time and, for
// a hand-built CallInterface, at call time. The function is never called.
func TestStackLayout_TooManyWordsIsAnError(t *testing.T) {
	requireStructLib(t)
	sym, err := GetSymbol(structTestLib, "stack_i64x16")
	if err != nil {
		t.Fatal(err)
	}
	const n = 1 << 14 // 16384 int64: far more stack than any call path allows
	argTypes := repeatType(types.SInt64TypeDescriptor, n)
	var v int64
	args := make([]unsafe.Pointer, n)
	for i := range args {
		args[i] = unsafe.Pointer(&v)
	}

	var cif types.CallInterface
	err = PrepareCallInterface(&cif, types.DefaultCall, types.SInt64TypeDescriptor, argTypes)
	if !errors.Is(err, ErrTooManyArguments) {
		t.Errorf("PrepareCallInterface with %d arguments: err = %v, want ErrTooManyArguments", n, err)
	}

	hand := types.CallInterface{ReturnType: types.SInt64TypeDescriptor, ArgTypes: argTypes}
	var got int64
	if _, err := CallFunction(&hand, sym, unsafe.Pointer(&got), args); err == nil {
		t.Errorf("CallFunction with %d arguments on a hand-built CallInterface returned no error", n)
	}
}

// Callbacks decode the same layout: C passes the struct on the stack whole and
// the next argument in the register that was left.
func TestCallbackStackLayout(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOARCH == "arm64" {
		t.Skip("callback struct args not supported on Windows/ARM64")
	}
	requireStructLib(t)

	callPtr := func(t *testing.T, name string, ret *types.TypeDescriptor, rvalue unsafe.Pointer, cb uintptr) {
		t.Helper()
		if err := callStackFunc(t, name, ret, rvalue, []*types.TypeDescriptor{types.PointerTypeDescriptor},
			[]unsafe.Pointer{unsafe.Pointer(&cb)}); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("split s16 after 5 int64", func(t *testing.T) {
		type pairI64 struct{ A, B int64 }
		cb := NewCallback(func(r0, r1, r2, r3, r4 int64, s pairI64, last int64) int64 {
			return r0 + 2*r1 + 3*r2 + 4*r3 + 5*r4 + 1000*s.A + 100*s.B + 100000*last
		})
		want := int64(1+4+9+16+25) + 1000*42 + 100*7 + 100000*9
		var got int64
		callPtr(t, "cb_call_split_s16", types.SInt64TypeDescriptor, unsafe.Pointer(&got), cb)
		if got != want {
			t.Errorf("callback result = %d, want %d", got, want)
		}
	})

	t.Run("split {double,double} after 7 doubles", func(t *testing.T) {
		type pairF64 struct{ A, B float64 }
		cb := NewCallback(func(d0, d1, d2, d3, d4, d5, d6 float64, s pairF64, last float64) float64 {
			return d0 + 2*d1 + 3*d2 + 4*d3 + 5*d4 + 6*d5 + 7*d6 + 1000*s.A + 100*s.B + 100000*last
		})
		want := float64(1+4+9+16+25+36+49) + 1000*42 + 100*7 + 100000*9
		var got float64
		callPtr(t, "cb_call_split_d2", types.DoubleTypeDescriptor, unsafe.Pointer(&got), cb)
		if got != want {
			t.Errorf("callback result = %v, want %v", got, want)
		}
	})

	t.Run("96-byte struct", func(t *testing.T) {
		cb := NewCallback(func(s adapterInfo96) uint64 { return s.sum() })
		want := newAdapterInfo96(0)
		var got uint64
		callPtr(t, "cb_call_info96", u64Type, unsafe.Pointer(&got), cb)
		if got != want.sum() {
			t.Errorf("callback result = %d, want %d", got, want.sum())
		}
	})

	t.Run("nested float pairs", func(t *testing.T) {
		cb := NewCallback(func(s quadF32) float32 { return s.A.X + 2*s.A.Y + 3*s.B.X + 4*s.B.Y })
		var got float32
		callPtr(t, "cb_call_nested_f32x4", types.FloatTypeDescriptor, unsafe.Pointer(&got), cb)
		if want := float32(1 + 4 + 9 + 16); got != want {
			t.Errorf("callback result = %v, want %v", got, want)
		}
	})
}

type pairF32 struct{ X, Y float32 }
type quadF32 struct{ A, B pairF32 }

// quadF32Type describes struct quad_f32 {struct pair_f32 a, b;} with nested
// member descriptors, as a binding generator emits them.
func quadF32Type() *types.TypeDescriptor {
	pair := &types.TypeDescriptor{
		Kind: types.StructType, Size: 8, Alignment: 4,
		Members: []*types.TypeDescriptor{types.FloatTypeDescriptor, types.FloatTypeDescriptor},
	}
	return &types.TypeDescriptor{
		Kind: types.StructType, Size: 16, Alignment: 4,
		Members: []*types.TypeDescriptor{pair, pair},
	}
}

// Nested structs are classified by their scalar members: two {float, float}
// pairs are two SSE eightbytes on System V amd64, not INTEGER.
func TestStackLayout_NestedFloatStruct(t *testing.T) {
	t.Run("argument", func(t *testing.T) {
		s := quadF32{A: pairF32{1.5, 2.5}, B: pairF32{3.5, 4.5}}
		var got float32
		if err := callStackFunc(t, "take_nested_f32x4", types.FloatTypeDescriptor, unsafe.Pointer(&got),
			[]*types.TypeDescriptor{quadF32Type()}, []unsafe.Pointer{unsafe.Pointer(&s)}); err != nil {
			t.Fatal(err)
		}
		if want := s.A.X + 2*s.A.Y + 3*s.B.X + 4*s.B.Y; got != want {
			t.Errorf("take_nested_f32x4 = %v, want %v", got, want)
		}
	})

	t.Run("result", func(t *testing.T) {
		in := [4]float32{1.5, 2.5, 3.5, 4.5}
		args := []unsafe.Pointer{unsafe.Pointer(&in[0]), unsafe.Pointer(&in[1]),
			unsafe.Pointer(&in[2]), unsafe.Pointer(&in[3])}
		var got quadF32
		if err := callStackFunc(t, "return_nested_f32x4", quadF32Type(), unsafe.Pointer(&got),
			repeatType(types.FloatTypeDescriptor, 4), args); err != nil {
			t.Fatal(err)
		}
		if want := (quadF32{A: pairF32{1.5, 2.5}, B: pairF32{3.5, 4.5}}); got != want {
			t.Errorf("return_nested_f32x4 = %+v, want %+v", got, want)
		}
	})
}
