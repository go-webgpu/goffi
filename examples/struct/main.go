// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// Package main demonstrates struct pass-by-value and struct return via goffi.
//
// It compiles structlib.c at runtime using gcc, loads the resulting shared
// library, and exercises four C functions that cover the three struct ABI
// size classes: ≤16 B (register pairs), >16 B (sret hidden pointer).
//
// Run:
//
//	go run . (requires gcc in PATH)
//
// The example gracefully skips when gcc is unavailable.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
)

// Point mirrors C: typedef struct { int64_t x; int64_t y; } Point;
type Point struct {
	X int64
	Y int64
}

// Vec3 mirrors C: typedef struct { int64_t x; int64_t y; int64_t z; } Vec3;
type Vec3 struct {
	X int64
	Y int64
	Z int64
}

func main() {
	libPath, ok := buildLib()
	if !ok {
		fmt.Println("gcc not found — skipping struct example (install gcc to run)")
		return
	}

	handle, err := ffi.LoadLibrary(libPath)
	if err != nil {
		fmt.Println("LoadLibrary error:", err)
		return
	}
	defer ffi.FreeLibrary(handle)

	// TypeDescriptor for Point {int64, int64} — 16 bytes, alignment 8.
	// Size and Alignment must match C sizeof/alignof exactly.
	pointType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.SInt64TypeDescriptor, // x
			types.SInt64TypeDescriptor, // y
		},
	}

	// TypeDescriptor for Vec3 {int64, int64, int64} — 24 bytes (>16 B → sret).
	vec3Type := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      24,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.SInt64TypeDescriptor, // x
			types.SInt64TypeDescriptor, // y
			types.SInt64TypeDescriptor, // z
		},
	}

	demoMakePoint(handle, pointType)
	demoDistanceSquared(handle, pointType)
	demoMakeVec3(handle, vec3Type)
	demoVec3Dot(handle, vec3Type)
}

// demoMakePoint calls: Point make_point(int64_t x, int64_t y)
// Return: ≤16 B struct — goffi reads it from GP register pair (Unix) or sret (Windows).
func demoMakePoint(handle unsafe.Pointer, pointType *types.TypeDescriptor) {
	sym, err := ffi.GetSymbol(handle, "make_point")
	if err != nil {
		fmt.Println("GetSymbol make_point error:", err)
		return
	}

	var cif types.CallInterface
	if err = ffi.PrepareCallInterface(&cif, types.DefaultCall,
		pointType,
		[]*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
		},
	); err != nil {
		fmt.Println("PrepareCallInterface error:", err)
		return
	}

	x, y := int64(3), int64(4)
	var result Point
	if _, err = ffi.CallFunction(&cif, sym,
		unsafe.Pointer(&result), // buffer for struct return value
		[]unsafe.Pointer{
			unsafe.Pointer(&x),
			unsafe.Pointer(&y),
		},
	); err != nil {
		fmt.Println("CallFunction make_point error:", err)
		return
	}

	fmt.Printf("make_point(%d, %d) = {X:%d Y:%d}\n", x, y, result.X, result.Y)
}

// demoDistanceSquared calls: int64_t distance_squared(Point a, Point b)
// Arguments: two Point structs passed by value.
func demoDistanceSquared(handle unsafe.Pointer, pointType *types.TypeDescriptor) {
	sym, err := ffi.GetSymbol(handle, "distance_squared")
	if err != nil {
		fmt.Println("GetSymbol distance_squared error:", err)
		return
	}

	var cif types.CallInterface
	if err = ffi.PrepareCallInterface(&cif, types.DefaultCall,
		types.SInt64TypeDescriptor,
		[]*types.TypeDescriptor{pointType, pointType}, // two Point args
	); err != nil {
		fmt.Println("PrepareCallInterface error:", err)
		return
	}

	a := Point{X: 0, Y: 0}
	b := Point{X: 3, Y: 4}
	var dist int64
	if _, err = ffi.CallFunction(&cif, sym,
		unsafe.Pointer(&dist),
		[]unsafe.Pointer{
			unsafe.Pointer(&a), // pointer to struct data
			unsafe.Pointer(&b),
		},
	); err != nil {
		fmt.Println("CallFunction distance_squared error:", err)
		return
	}

	// distance_squared({0,0}, {3,4}) = 3²+4² = 25
	fmt.Printf("distance_squared({%d,%d}, {%d,%d}) = %d\n",
		a.X, a.Y, b.X, b.Y, dist)
}

// demoMakeVec3 calls: Vec3 make_vec3(int64_t x, int64_t y, int64_t z)
// Return: >16 B struct — always via sret hidden pointer; goffi handles this transparently.
func demoMakeVec3(handle unsafe.Pointer, vec3Type *types.TypeDescriptor) {
	sym, err := ffi.GetSymbol(handle, "make_vec3")
	if err != nil {
		fmt.Println("GetSymbol make_vec3 error:", err)
		return
	}

	var cif types.CallInterface
	if err = ffi.PrepareCallInterface(&cif, types.DefaultCall,
		vec3Type,
		[]*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
		},
	); err != nil {
		fmt.Println("PrepareCallInterface error:", err)
		return
	}

	x, y, z := int64(1), int64(2), int64(3)
	var result Vec3
	if _, err = ffi.CallFunction(&cif, sym,
		unsafe.Pointer(&result), // goffi passes &result as the hidden sret pointer
		[]unsafe.Pointer{
			unsafe.Pointer(&x),
			unsafe.Pointer(&y),
			unsafe.Pointer(&z),
		},
	); err != nil {
		fmt.Println("CallFunction make_vec3 error:", err)
		return
	}

	fmt.Printf("make_vec3(%d, %d, %d) = {X:%d Y:%d Z:%d}\n",
		x, y, z, result.X, result.Y, result.Z)
}

// demoVec3Dot calls: int64_t vec3_dot(Vec3 a, Vec3 b)
// Arguments: two Vec3 structs (>16 B each) passed by value on the stack.
func demoVec3Dot(handle unsafe.Pointer, vec3Type *types.TypeDescriptor) {
	sym, err := ffi.GetSymbol(handle, "vec3_dot")
	if err != nil {
		fmt.Println("GetSymbol vec3_dot error:", err)
		return
	}

	var cif types.CallInterface
	if err = ffi.PrepareCallInterface(&cif, types.DefaultCall,
		types.SInt64TypeDescriptor,
		[]*types.TypeDescriptor{vec3Type, vec3Type},
	); err != nil {
		fmt.Println("PrepareCallInterface error:", err)
		return
	}

	a := Vec3{X: 1, Y: 2, Z: 3}
	b := Vec3{X: 4, Y: 5, Z: 6}
	var dot int64
	if _, err = ffi.CallFunction(&cif, sym,
		unsafe.Pointer(&dot),
		[]unsafe.Pointer{
			unsafe.Pointer(&a),
			unsafe.Pointer(&b),
		},
	); err != nil {
		fmt.Println("CallFunction vec3_dot error:", err)
		return
	}

	// dot = 1*4 + 2*5 + 3*6 = 4 + 10 + 18 = 32
	fmt.Printf("vec3_dot({%d,%d,%d}, {%d,%d,%d}) = %d\n",
		a.X, a.Y, a.Z, b.X, b.Y, b.Z, dot)
}

// buildLib compiles structlib.c into a shared library and returns its path.
// Returns ("", false) if gcc is not available.
func buildLib() (string, bool) {
	cc := os.Getenv("CC")
	if cc == "" {
		cc = "gcc"
	}
	if _, err := exec.LookPath(cc); err != nil {
		return "", false
	}

	dir, err := os.MkdirTemp("", "goffi-struct-example-*")
	if err != nil {
		fmt.Println("TempDir error:", err)
		return "", false
	}

	src := filepath.Join(filepath.Dir(os.Args[0]), "structlib.c")
	if _, statErr := os.Stat(src); statErr != nil {
		// When invoked via "go run .", the source directory is the working directory.
		src = "structlib.c"
	}

	var soPath string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		soPath = filepath.Join(dir, "libstructlib.dylib")
		args = []string{"-shared", "-fPIC", "-O2", "-o", soPath, src}
	case "windows":
		soPath = filepath.Join(dir, "structlib.dll")
		args = []string{"-shared", "-O2", "-o", soPath, src}
	default:
		soPath = filepath.Join(dir, "libstructlib.so")
		args = []string{"-shared", "-fPIC", "-O2", "-o", soPath, src}
	}

	cmd := exec.Command(cc, args...)
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		fmt.Println("gcc compile error:", err)
		return "", false
	}

	return soPath, true
}
