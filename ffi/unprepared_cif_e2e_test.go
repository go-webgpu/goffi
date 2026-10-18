// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux || darwin || freebsd || windows) && (amd64 || arm64)

package ffi

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// A types.CallInterface can be filled in as a struct literal instead of by
// PrepareCallInterface: Convention, Flags and StackBytes are then zero. The
// argument layout is recomputed from ArgTypes on every call, but how a struct
// result comes back depends on Flags — the register pair of a 9-16 byte struct
// on System V amd64, HFA and indirect (X8) results on arm64. These tests call
// such hand-built CallInterfaces and compare the result with what the C
// function returned.

// handBuiltCIF returns a CallInterface the way a caller would write it without
// PrepareCallInterface.
func handBuiltCIF(ret *types.TypeDescriptor, args ...*types.TypeDescriptor) *types.CallInterface {
	return &types.CallInterface{ArgCount: len(args), ArgTypes: args, ReturnType: ret}
}

// ptrTo returns a pointer to a copy of v, for avalue entries.
func ptrTo[T any](v T) unsafe.Pointer { return unsafe.Pointer(&v) }

func structDesc(size, align uintptr, members ...*types.TypeDescriptor) *types.TypeDescriptor {
	return &types.TypeDescriptor{Kind: types.StructType, Size: size, Alignment: align, Members: members}
}

// callRecover reports a panic inside CallFunction as an error, so a broken
// call fails its subtest instead of the whole test binary.
func callRecover(cif *types.CallInterface, fn, rvalue unsafe.Pointer, avalue []unsafe.Pointer) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	_, err = CallFunction(cif, fn, rvalue, avalue)
	return err
}

func TestUnpreparedCallInterface_StructReturn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Win64 returns 16- and 24-byte structs through a hidden result pointer; covered by the Windows struct return tests")
	}
	requireStructLib(t)

	type pairF64 struct{ A, B float64 }
	type intFloat struct {
		A int64
		B float64
	}
	type floatInt struct {
		A float64
		B int64
	}
	type pairI64 struct{ A, B int64 }
	type tripleI64 struct{ A, B, C int64 }

	i64, f64 := types.SInt64TypeDescriptor, types.DoubleTypeDescriptor

	// The 24-byte case goes last: without an indirect result pointer the
	// callee writes through whatever X8 holds.
	tests := []struct {
		name string
		sym  string
		ret  *types.TypeDescriptor
		args []*types.TypeDescriptor
		vals []unsafe.Pointer
		got  func(rvalue unsafe.Pointer) any
		want any
		buf  unsafe.Pointer
	}{
		{
			// {SSE, SSE} on System V (XMM0:XMM1); HFA of two doubles on arm64 (D0:D1).
			name: "two doubles", sym: "return_struct_2doubles",
			ret: structDesc(16, 8, f64, f64), args: []*types.TypeDescriptor{f64, f64},
			vals: []unsafe.Pointer{ptrTo(1.5), ptrTo(-2.25)},
			buf:  unsafe.Pointer(&pairF64{}),
			got:  func(p unsafe.Pointer) any { return *(*pairF64)(p) },
			want: pairF64{1.5, -2.25},
		},
		{
			// {INTEGER, SSE}: RAX:XMM0 on System V; X0:X1 on arm64.
			name: "int64 and double", sym: "return_struct_int_float",
			ret: structDesc(16, 8, i64, f64), args: []*types.TypeDescriptor{i64, f64},
			vals: []unsafe.Pointer{ptrTo(int64(-7)), ptrTo(3.5)},
			buf:  unsafe.Pointer(&intFloat{}),
			got:  func(p unsafe.Pointer) any { return *(*intFloat)(p) },
			want: intFloat{-7, 3.5},
		},
		{
			// {SSE, INTEGER}: XMM0:RAX on System V; X0:X1 on arm64.
			name: "double and int64", sym: "return_struct_float_int",
			ret: structDesc(16, 8, f64, i64), args: []*types.TypeDescriptor{f64, i64},
			vals: []unsafe.Pointer{ptrTo(4.75), ptrTo(int64(9))},
			buf:  unsafe.Pointer(&floatInt{}),
			got:  func(p unsafe.Pointer) any { return *(*floatInt)(p) },
			want: floatInt{4.75, 9},
		},
		{
			// {INTEGER, INTEGER}: RAX:RDX on System V; X0:X1 on arm64.
			name: "two int64", sym: "return_struct_2ints",
			ret: structDesc(16, 8, i64, i64), args: []*types.TypeDescriptor{i64, i64},
			vals: []unsafe.Pointer{ptrTo(int64(5)), ptrTo(int64(-6))},
			buf:  unsafe.Pointer(&pairI64{}),
			got:  func(p unsafe.Pointer) any { return *(*pairI64)(p) },
			want: pairI64{5, -6},
		},
		{
			// Larger than 16 bytes: hidden result pointer (RDI on System V, X8 on arm64).
			name: "24 bytes", sym: "return_struct_24",
			ret:  structDesc(24, 8, i64, i64, i64),
			buf:  unsafe.Pointer(&tripleI64{}),
			got:  func(p unsafe.Pointer) any { return *(*tripleI64)(p) },
			want: tripleI64{11, 22, 33},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sym, err := GetSymbol(structTestLib, tt.sym)
			if err != nil {
				t.Fatal(err)
			}
			cif := handBuiltCIF(tt.ret, tt.args...)
			if err := callRecover(cif, sym, tt.buf, tt.vals); err != nil {
				t.Fatalf("%s through a hand-built CallInterface: %v", tt.sym, err)
			}
			if got := tt.got(tt.buf); got != tt.want {
				t.Errorf("%s through a hand-built CallInterface = %+v, want %+v", tt.sym, got, tt.want)
			}
			// The call must not have written the caller's CallInterface: it may
			// be shared between goroutines.
			if cif.Flags != 0 || cif.Convention != types.DefaultCall || cif.StackBytes != 0 {
				t.Errorf("CallFunction modified the CallInterface: Convention=%d Flags=%d StackBytes=%d",
					cif.Convention, cif.Flags, cif.StackBytes)
			}
		})
	}
}

// One hand-built CallInterface shared by many goroutines, as a package-level
// variable would be. Run with -race to catch writes to it.
func TestUnpreparedCallInterface_SharedBetweenGoroutines(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Win64 returns 16-byte structs through a hidden result pointer; covered by the Windows struct return tests")
	}
	requireStructLib(t)
	sym, err := GetSymbol(structTestLib, "return_struct_2doubles")
	if err != nil {
		t.Fatal(err)
	}
	f64 := types.DoubleTypeDescriptor
	cif := handBuiltCIF(structDesc(16, 8, f64, f64), f64, f64)

	const goroutines, calls = 8, 200
	errs := make(chan error, goroutines)
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range calls {
				a, b := float64(g), float64(i)+0.5
				var out struct{ A, B float64 }
				if err := callRecover(cif, sym, unsafe.Pointer(&out),
					[]unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b)}); err != nil {
					errs <- err
					return
				}
				if out.A != a || out.B != b {
					errs <- fmt.Errorf("goroutine %d call %d: got {%v, %v}, want {%v, %v}", g, i, out.A, out.B, a, b)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// A struct result whose size was never computed cannot be classified without
// writing the caller's descriptor; the call must fail instead of guessing.
func TestUnpreparedCallInterface_StructSizeNotSet(t *testing.T) {
	requireStructLib(t)
	sym, err := GetSymbol(structTestLib, "return_struct_2ints")
	if err != nil {
		t.Fatal(err)
	}
	i64 := types.SInt64TypeDescriptor
	ret := &types.TypeDescriptor{Kind: types.StructType, Members: []*types.TypeDescriptor{i64, i64}}
	cif := handBuiltCIF(ret, i64, i64)

	a, b := int64(1), int64(2)
	var out [2]int64
	err = callRecover(cif, sym, unsafe.Pointer(&out), []unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b)})
	var cifErr *InvalidCallInterfaceError
	if !errors.As(err, &cifErr) || cifErr.Field != "returnType" {
		t.Fatalf("CallFunction with an unsized struct result: err = %v, want *InvalidCallInterfaceError for returnType", err)
	}
	if ret.Size != 0 || ret.Alignment != 0 {
		t.Errorf("CallFunction wrote the return type descriptor: Size=%d Alignment=%d", ret.Size, ret.Alignment)
	}
}

func TestUnpreparedCallInterface_NilReturnType(t *testing.T) {
	requireStructLib(t)
	sym, err := GetSymbol(structTestLib, "int_weighted3")
	if err != nil {
		t.Fatal(err)
	}
	it := types.IntTypeDescriptor
	cif := handBuiltCIF(nil, it, it, it)

	a, b, c := int32(1), int32(2), int32(3)
	err = callRecover(cif, sym, nil, []unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b), unsafe.Pointer(&c)})
	var cifErr *InvalidCallInterfaceError
	if !errors.As(err, &cifErr) || cifErr.Field != "returnType" {
		t.Fatalf("CallFunction with a nil return type: err = %v, want *InvalidCallInterfaceError for returnType", err)
	}
}

// Scalar results do not depend on Flags: a hand-built CallInterface works as it
// always has, with arguments on the stack included.
func TestUnpreparedCallInterface_ScalarReturn(t *testing.T) {
	requireStructLib(t)
	sym, err := GetSymbol(structTestLib, "int_weighted3")
	if err != nil {
		t.Fatal(err)
	}
	it := types.IntTypeDescriptor
	cif := handBuiltCIF(it, it, it, it)

	a, b, c := int32(-4), int32(5), int32(-6)
	var got int32
	if err := callRecover(cif, sym, unsafe.Pointer(&got),
		[]unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b), unsafe.Pointer(&c)}); err != nil {
		t.Fatal(err)
	}
	if want := a + 2*b + 3*c; got != want {
		t.Errorf("int_weighted3(%d, %d, %d) = %d, want %d", a, b, c, got, want)
	}
}

// CallFunction tells a hand-built CallInterface from a prepared one by
// Flags == 0 with a struct result. That only works if PrepareCallInterface
// never leaves Flags at zero for a struct result.
func TestPrepareCallInterface_StructReturnFlagsNonZero(t *testing.T) {
	u8, f32, f64, i64 := types.UInt8TypeDescriptor, types.FloatTypeDescriptor, types.DoubleTypeDescriptor, types.SInt64TypeDescriptor
	repeat := func(n int, m *types.TypeDescriptor) []*types.TypeDescriptor {
		ms := make([]*types.TypeDescriptor, n)
		for i := range ms {
			ms[i] = m
		}
		return ms
	}
	var cases []*types.TypeDescriptor
	for n := 1; n <= 33; n++ {
		cases = append(cases, &types.TypeDescriptor{Kind: types.StructType, Members: repeat(n, u8)})
	}
	for n := 1; n <= 5; n++ {
		cases = append(cases,
			&types.TypeDescriptor{Kind: types.StructType, Members: repeat(n, f32)},
			&types.TypeDescriptor{Kind: types.StructType, Members: repeat(n, f64)},
			&types.TypeDescriptor{Kind: types.StructType, Members: repeat(n, i64)})
	}
	cases = append(cases,
		&types.TypeDescriptor{Kind: types.StructType, Members: []*types.TypeDescriptor{f32, i64}},
		&types.TypeDescriptor{Kind: types.StructType, Members: []*types.TypeDescriptor{i64, f32}},
		&types.TypeDescriptor{Kind: types.StructType, Members: []*types.TypeDescriptor{
			{Kind: types.StructType, Members: []*types.TypeDescriptor{f64, f64}}, f64}})

	for _, rt := range cases {
		var cif types.CallInterface
		if err := PrepareCallInterface(&cif, types.DefaultCall, rt, nil); err != nil {
			t.Fatalf("PrepareCallInterface(struct of %d members, size %d): %v", len(rt.Members), rt.Size, err)
		}
		if cif.Flags == 0 {
			t.Errorf("PrepareCallInterface left Flags = 0 for a struct result of %d members, size %d",
				len(rt.Members), rt.Size)
		}
	}
}
