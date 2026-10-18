// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux || darwin || freebsd || windows) && (amd64 || arm64)

package ffi

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// Structs of every size class, against return_bytesN / take_bytesN from
// testdata/structtest.c (N uint8_t members). Win64 passes and returns only
// 1, 2, 4 and 8 byte structs in registers; any other size is returned through
// a hidden result pointer and passed as a pointer to a caller-owned copy.
// System V amd64 and AAPCS64 use registers up to 16 bytes and memory above.
var structSizes = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 15, 16, 17, 24, 32}

func bytesStruct(n int) *types.TypeDescriptor {
	members := make([]*types.TypeDescriptor, n)
	for i := range members {
		members[i] = types.UInt8TypeDescriptor
	}
	return &types.TypeDescriptor{Kind: types.StructType, Members: members}
}

func bytesPattern(n int, seed uint8) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = seed + uint8(i*7)
	}
	return b
}

const canary = 0xCC

// The result must be exactly the struct: every byte right, nothing written
// past it (rvalue is the caller's struct, not a register-sized buffer).
func TestStructBySize_Return(t *testing.T) {
	requireStructLib(t)
	for _, n := range structSizes {
		t.Run(fmt.Sprintf("%dB", n), func(t *testing.T) {
			sym, err := GetSymbol(structTestLib, fmt.Sprintf("return_bytes%d", n))
			if err != nil {
				t.Fatal(err)
			}
			var cif types.CallInterface
			if err := PrepareCallInterface(&cif, types.DefaultCall, bytesStruct(n),
				[]*types.TypeDescriptor{types.UInt8TypeDescriptor}); err != nil {
				t.Fatal(err)
			}
			buf := make([]byte, n+16)
			for i := range buf {
				buf[i] = canary
			}
			seed := uint8(n * 3)
			if _, err := CallFunction(&cif, sym, unsafe.Pointer(&buf[0]), []unsafe.Pointer{unsafe.Pointer(&seed)}); err != nil {
				t.Fatal(err)
			}
			if want := bytesPattern(n, seed); string(buf[:n]) != string(want) {
				t.Errorf("return_bytes%d(%d) = % x, want % x", n, seed, buf[:n], want)
			}
			for i := n; i < len(buf); i++ {
				if buf[i] != canary {
					t.Errorf("return_bytes%d wrote past the %d-byte result: byte %d = %#x", n, n, i, buf[i])
					break
				}
			}
		})
	}
}

// The callee receives the struct by value and overwrites its copy; the
// caller's value must not change.
func TestStructBySize_Arg(t *testing.T) {
	requireStructLib(t)
	for _, n := range structSizes {
		t.Run(fmt.Sprintf("%dB", n), func(t *testing.T) {
			sym, err := GetSymbol(structTestLib, fmt.Sprintf("take_bytes%d", n))
			if err != nil {
				t.Fatal(err)
			}
			var cif types.CallInterface
			if err := PrepareCallInterface(&cif, types.DefaultCall, types.SInt64TypeDescriptor,
				[]*types.TypeDescriptor{bytesStruct(n)}); err != nil {
				t.Fatal(err)
			}
			val := bytesPattern(n, uint8(n*5+1))
			orig := string(val)
			var want int64
			for _, b := range val {
				want = want*31 + int64(b)
			}
			var got int64
			if _, err := CallFunction(&cif, sym, unsafe.Pointer(&got), []unsafe.Pointer{unsafe.Pointer(&val[0])}); err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("take_bytes%d(% x) = %d, want %d", n, []byte(orig), got, want)
			}
			if string(val) != orig {
				t.Errorf("take_bytes%d changed the caller's argument: % x, was % x", n, val, []byte(orig))
			}
		})
	}
}

// Small structs of floating-point members come back in floating-point
// registers on System V amd64 (an all-float eightbyte is SSE class) and on
// arm64 (an HFA, a single-member one included), in RAX on Win64.
func TestStructReturn_FloatMembers(t *testing.T) {
	requireStructLib(t)
	f32, f64, i32 := types.FloatTypeDescriptor, types.DoubleTypeDescriptor, types.SInt32TypeDescriptor

	type oneF32 struct{ A float32 }
	type oneF64 struct{ A float64 }
	type pairF32 struct{ A, B float32 }
	type threeF32 struct{ A, B, C float32 }
	type f32I32 struct {
		A float32
		B int32
	}

	tests := []struct {
		sym     string
		members []*types.TypeDescriptor
		args    []*types.TypeDescriptor
		vals    []unsafe.Pointer
		size    uintptr
		got     func(unsafe.Pointer) any
		want    any
	}{
		{"return_struct_1float", []*types.TypeDescriptor{f32}, []*types.TypeDescriptor{f32},
			[]unsafe.Pointer{ptrTo(float32(1.25))}, 4,
			func(p unsafe.Pointer) any { return *(*oneF32)(p) }, oneF32{1.25}},
		{"return_struct_1double", []*types.TypeDescriptor{f64}, []*types.TypeDescriptor{f64},
			[]unsafe.Pointer{ptrTo(-6.5)}, 8,
			func(p unsafe.Pointer) any { return *(*oneF64)(p) }, oneF64{-6.5}},
		{"return_struct_2floats", []*types.TypeDescriptor{f32, f32}, []*types.TypeDescriptor{f32, f32},
			[]unsafe.Pointer{ptrTo(float32(2.5)), ptrTo(float32(-3.75))}, 8,
			func(p unsafe.Pointer) any { return *(*pairF32)(p) }, pairF32{2.5, -3.75}},
		{"return_struct_3floats", []*types.TypeDescriptor{f32, f32, f32}, []*types.TypeDescriptor{f32, f32, f32},
			[]unsafe.Pointer{ptrTo(float32(1)), ptrTo(float32(2)), ptrTo(float32(3))}, 12,
			func(p unsafe.Pointer) any { return *(*threeF32)(p) }, threeF32{1, 2, 3}},
		{"return_struct_float_int32", []*types.TypeDescriptor{f32, i32}, []*types.TypeDescriptor{f32, i32},
			[]unsafe.Pointer{ptrTo(float32(0.5)), ptrTo(int32(-9))}, 8,
			func(p unsafe.Pointer) any { return *(*f32I32)(p) }, f32I32{0.5, -9}},
	}
	for _, tt := range tests {
		t.Run(tt.sym, func(t *testing.T) {
			sym, err := GetSymbol(structTestLib, tt.sym)
			if err != nil {
				t.Fatal(err)
			}
			var cif types.CallInterface
			ret := &types.TypeDescriptor{Kind: types.StructType, Members: tt.members}
			if err := PrepareCallInterface(&cif, types.DefaultCall, ret, tt.args); err != nil {
				t.Fatal(err)
			}
			if ret.Size != tt.size {
				t.Fatalf("struct size = %d, want %d", ret.Size, tt.size)
			}
			buf := make([]byte, tt.size+8)
			for i := range buf {
				buf[i] = canary
			}
			if _, err := CallFunction(&cif, sym, unsafe.Pointer(&buf[0]), tt.vals); err != nil {
				t.Fatal(err)
			}
			if got := tt.got(unsafe.Pointer(&buf[0])); got != tt.want {
				t.Errorf("%s = %+v, want %+v", tt.sym, got, tt.want)
			}
			for i := tt.size; i < uintptr(len(buf)); i++ {
				if buf[i] != canary {
					t.Errorf("%s wrote past the %d-byte result: byte %d = %#x", tt.sym, tt.size, i, buf[i])
					break
				}
			}
		})
	}
}

// libffi accepts a nil rvalue for any result and discards it; for a struct
// returned through a hidden pointer the callee still needs somewhere to write.
// The calls run in a child process so that a crash fails this test only.
func TestStructBySize_NilResult(t *testing.T) {
	requireStructLib(t)
	if os.Getenv("GOFFI_NIL_RESULT_CHILD") == "1" {
		for _, n := range structSizes {
			sym, err := GetSymbol(structTestLib, fmt.Sprintf("return_bytes%d", n))
			if err != nil {
				t.Fatal(err)
			}
			var cif types.CallInterface
			if err := PrepareCallInterface(&cif, types.DefaultCall, bytesStruct(n),
				[]*types.TypeDescriptor{types.UInt8TypeDescriptor}); err != nil {
				t.Fatal(err)
			}
			seed := uint8(n)
			if _, err := CallFunction(&cif, sym, nil, []unsafe.Pointer{unsafe.Pointer(&seed)}); err != nil {
				t.Errorf("return_bytes%d with a nil rvalue: %v", n, err)
			}
			fmt.Printf("nil-result %dB ok\n", n)
		}
		return
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestStructBySize_NilResult$", "-test.count=1")
	cmd.Env = append(os.Environ(), "GOFFI_NIL_RESULT_CHILD=1", structTestLibEnv+"="+structTestLibPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("calls with a nil rvalue failed: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	if !strings.Contains(string(out), fmt.Sprintf("nil-result %dB ok", structSizes[len(structSizes)-1])) {
		t.Fatalf("child process did not run all sizes:\n%s", out)
	}
}
