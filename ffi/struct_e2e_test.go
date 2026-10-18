// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build (linux || darwin || freebsd || windows) && (amd64 || arm64)

package ffi

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

var (
	structTestLib unsafe.Pointer
	// structTestLibPath is the absolute path of the loaded library.
	structTestLibPath string
	// structTestLibSkip says why the C-backed tests skip; empty when the
	// library was built and loaded.
	structTestLibSkip string
)

// structTestLibEnv names a library a parent test process already built; a
// child process (see TestStructBySize_NilResult) loads it instead of
// rebuilding, which on Windows would fail while the parent holds the DLL.
const structTestLibEnv = "GOFFI_STRUCTTEST_LIB"

// errNoCCompiler reports that there is no C compiler for the platform the
// tests run on: none installed, or one that targets another architecture.
var errNoCCompiler = errors.New("no C compiler for this target")

// The C-backed end-to-end tests (struct, variadic, callback float ABI) build
// testdata/structtest.c and load it. They skip only for a reason that makes
// running them impossible here; anything else that keeps the library from
// loading is a test failure, never a skip.
func TestMain(m *testing.M) {
	if runtime.GOOS == "android" {
		// Android test binaries run on-device, where invoking a host compiler
		// is neither meaningful nor available.
		structTestLibSkip = "android tests run on-device without a C compiler"
	} else {
		var err error
		if path := os.Getenv(structTestLibEnv); path != "" {
			err = loadStructTestLib(path)
		} else {
			err = buildStructTestLib()
		}
		switch {
		case err == nil:
		case errors.Is(err, ErrStaticBuild):
			structTestLibSkip = "dynamic loading is unavailable in this build (-tags goffi_static)"
		case errors.Is(err, errNoCCompiler) && os.Getenv("GOFFI_REQUIRE_TESTLIB") != "1":
			structTestLibSkip = err.Error()
		default:
			// CI sets GOFFI_REQUIRE_TESTLIB=1 on every runner that has a
			// compiler for its own architecture, so losing it fails the job.
			fmt.Fprintf(os.Stderr, "ffi tests: C test library unavailable: %v\n", err)
			os.Exit(1)
		}
	}
	code := m.Run()
	if structTestLib != nil {
		FreeLibrary(structTestLib)
	}
	os.Exit(code)
}

func buildStructTestLib() error {
	srcPath := filepath.Join("testdata", "structtest.c")
	var soPath string
	switch runtime.GOOS {
	case "darwin":
		soPath = filepath.Join("testdata", "libstructtest.dylib")
	case "windows":
		soPath = filepath.Join("testdata", "structtest.dll")
	default:
		soPath = filepath.Join("testdata", "libstructtest.so")
	}

	cc := os.Getenv("CC")
	if cc == "" {
		cc = "gcc"
	}
	ccPath, err := exec.LookPath(cc)
	if err != nil {
		return fmt.Errorf("%w (set CC): %v", errNoCCompiler, err)
	}
	// A compiler for another architecture cannot produce a library this
	// process can load (windows-11-arm runners ship an x86-64 MinGW gcc).
	if triple := ccTargetTriple(ccPath); triple != "" && !tripleMatchesGOARCH(triple) {
		return fmt.Errorf("%w (set CC): %s targets %s, the tests run on %s/%s",
			errNoCCompiler, cc, triple, runtime.GOOS, runtime.GOARCH)
	}
	args := []string{"-shared", "-O2", "-o", soPath, srcPath}
	if runtime.GOOS != "windows" {
		args = []string{"-shared", "-fPIC", "-O2", "-o", soPath, srcPath}
	}
	cmd := exec.Command(ccPath, args...)
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", cc, srcPath, err)
	}

	absPath, err := filepath.Abs(soPath)
	if err != nil {
		return err
	}
	return loadStructTestLib(absPath)
}

func loadStructTestLib(path string) error {
	lib, err := LoadLibrary(path)
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}
	structTestLib = lib
	structTestLibPath = path
	return nil
}

// ccTargetTriple returns the target triple the C compiler reports
// (gcc and clang: -dumpmachine), or "" when it does not say.
func ccTargetTriple(ccPath string) string {
	out, err := exec.Command(ccPath, "-dumpmachine").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// tripleMatchesGOARCH reports whether a target triple such as
// "x86_64-linux-gnu" or "arm64-apple-darwin24.0.0" is for runtime.GOARCH.
func tripleMatchesGOARCH(triple string) bool {
	arch, _, _ := strings.Cut(triple, "-")
	switch runtime.GOARCH {
	case "amd64":
		return arch == "x86_64" || arch == "amd64"
	case "arm64":
		return arch == "aarch64" || arch == "arm64"
	}
	return true
}

func requireStructLib(t *testing.T) {
	t.Helper()
	if structTestLib == nil {
		t.Skip("C test library not available: " + structTestLibSkip)
	}
}

// TestStructArg8B_IntegerPair tests issue #33: struct {int32, uint32} passed by value.
func TestStructArg8B_IntegerPair(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "take_struct_8")
	if err != nil {
		t.Fatal(err)
	}

	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      8,
		Alignment: 4,
		Members: []*types.TypeDescriptor{
			types.SInt32TypeDescriptor,
			types.UInt32TypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.SInt64TypeDescriptor,
		[]*types.TypeDescriptor{structType}); err != nil {
		t.Fatal(err)
	}

	type Pair struct {
		A int32
		B uint32
	}
	s := Pair{A: 42, B: 19}
	args := []unsafe.Pointer{unsafe.Pointer(&s)}
	var result int64
	if _, err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	expected := int64(42)*1000 + int64(19)
	if result != expected {
		t.Errorf("take_struct_8({42, 19}) = %d, want %d", result, expected)
	}
}

// TestStructArg8B_FloatPair tests SSE classification: struct {float, float}.
func TestStructArg8B_FloatPair(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "take_struct_2floats")
	if err != nil {
		t.Fatal(err)
	}

	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      8,
		Alignment: 4,
		Members: []*types.TypeDescriptor{
			types.FloatTypeDescriptor,
			types.FloatTypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.FloatTypeDescriptor,
		[]*types.TypeDescriptor{structType}); err != nil {
		t.Fatal(err)
	}

	type PairF32 struct {
		X float32
		Y float32
	}
	s := PairF32{X: 2.5, Y: 3.5}
	args := []unsafe.Pointer{unsafe.Pointer(&s)}
	var result float32
	if _, err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	if result != 6.0 {
		t.Errorf("take_struct_2floats({2.5, 3.5}) = %f, want 6.0", result)
	}
}

// TestStructArg16B tests two-eightbyte struct: {int64, int64}.
func TestStructArg16B(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "take_struct_16")
	if err != nil {
		t.Fatal(err)
	}

	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      16,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.SInt64TypeDescriptor,
		[]*types.TypeDescriptor{structType}); err != nil {
		t.Fatal(err)
	}

	type PairI64 struct {
		A int64
		B int64
	}
	s := PairI64{A: 1000000, B: 2000000}
	args := []unsafe.Pointer{unsafe.Pointer(&s)}
	var result int64
	if _, err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	if result != 3000000 {
		t.Errorf("take_struct_16({1000000, 2000000}) = %d, want 3000000", result)
	}
}

// TestStructArg24B_MemoryClass tests > 16B struct passed on stack (MEMORY class).
func TestStructArg24B_MemoryClass(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "take_struct_24")
	if err != nil {
		t.Fatal(err)
	}

	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      24,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.SInt64TypeDescriptor,
		[]*types.TypeDescriptor{structType}); err != nil {
		t.Fatal(err)
	}

	type TripleI64 struct {
		A int64
		B int64
		C int64
	}
	s := TripleI64{A: 100, B: 200, C: 300}
	args := []unsafe.Pointer{unsafe.Pointer(&s)}
	var result int64
	if _, err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	if result != 600 {
		t.Errorf("take_struct_24({100, 200, 300}) = %d, want 600", result)
	}
}

// TestStructArgWithScalar tests struct + scalar argument (register allocation).
func TestStructArgWithScalar(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "take_struct_and_int")
	if err != nil {
		t.Fatal(err)
	}

	structType := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      8,
		Alignment: 4,
		Members: []*types.TypeDescriptor{
			types.SInt32TypeDescriptor,
			types.UInt32TypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.SInt64TypeDescriptor,
		[]*types.TypeDescriptor{structType, types.SInt64TypeDescriptor}); err != nil {
		t.Fatal(err)
	}

	type Pair struct {
		A int32
		B uint32
	}
	s := Pair{A: 10, B: 20}
	extra := int64(1000)
	args := []unsafe.Pointer{unsafe.Pointer(&s), unsafe.Pointer(&extra)}
	var result int64
	if _, err := CallFunction(&cif, sym, unsafe.Pointer(&result), args); err != nil {
		t.Fatal(err)
	}

	expected := int64(10) + int64(20) + int64(1000)
	if result != expected {
		t.Errorf("take_struct_and_int({10, 20}, 1000) = %d, want %d", result, expected)
	}
}

func TestCallbackStructArg8B_IntegerPair(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOARCH == "arm64" {
		t.Skip("callback struct args not supported on Windows/ARM64")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "callback_struct_8")
	if err != nil {
		t.Fatal(err)
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{
			types.SInt32TypeDescriptor,
			types.UInt32TypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatal(err)
	}

	type Pair struct {
		A int32
		B uint32
	}

	var receivedArg Pair
	callback := NewCallback(func(s Pair) {
		receivedArg = s
	})

	expected := Pair{42, 10}
	args := []unsafe.Pointer{
		unsafe.Pointer(&expected.A),
		unsafe.Pointer(&expected.B),
		unsafe.Pointer(&callback),
	}

	if _, err := CallFunction(&cif, sym, nil, args); err != nil {
		t.Fatal(err)
	}

	if receivedArg != expected {
		t.Errorf("expected %#v, received %#v", expected, receivedArg)
	}
}

func TestCallbackStructArg8B_FloatPair(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOARCH == "arm64" {
		t.Skip("callback struct args not supported on Windows/ARM64")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "callback_struct_2floats")
	if err != nil {
		t.Fatal(err)
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{
			types.FloatTypeDescriptor,
			types.FloatTypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatal(err)
	}

	type PairF32 struct {
		X float32
		Y float32
	}

	var receivedArg PairF32
	callback := NewCallback(func(s PairF32) {
		receivedArg = s
	})

	expected := PairF32{2.5, 3.5}
	args := []unsafe.Pointer{
		unsafe.Pointer(&expected.X),
		unsafe.Pointer(&expected.Y),
		unsafe.Pointer(&callback),
	}

	if _, err := CallFunction(&cif, sym, nil, args); err != nil {
		t.Fatal(err)
	}

	if receivedArg != expected {
		t.Errorf("expected %#v, received %#v", expected, receivedArg)
	}
}

func TestCallbackStructArg16B(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOARCH == "arm64" {
		t.Skip("callback struct args not supported on Windows/ARM64")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "callback_struct_16")
	if err != nil {
		t.Fatal(err)
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatal(err)
	}

	type PairI64 struct {
		A int64
		B int64
	}

	var receivedArg PairI64
	callback := NewCallback(func(s PairI64) {
		receivedArg = s
	})

	expected := PairI64{1000000, 2000000}
	args := []unsafe.Pointer{
		unsafe.Pointer(&expected.A),
		unsafe.Pointer(&expected.B),
		unsafe.Pointer(&callback),
	}

	if _, err := CallFunction(&cif, sym, nil, args); err != nil {
		t.Fatal(err)
	}

	if receivedArg != expected {
		t.Errorf("expected %#v, received %#v", expected, receivedArg)
	}
}

func TestCallbackStructArg24B_MemoryClass(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOARCH == "arm64" {
		t.Skip("callback struct args not supported on Windows/ARM64")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "callback_struct_24")
	if err != nil {
		t.Fatal(err)
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatal(err)
	}

	type TripleI64 struct {
		A int64
		B int64
		C int64
	}

	var receivedArg TripleI64
	callback := NewCallback(func(s TripleI64) {
		receivedArg = s
	})

	expected := TripleI64{100, 200, 300}
	args := []unsafe.Pointer{
		unsafe.Pointer(&expected.A),
		unsafe.Pointer(&expected.B),
		unsafe.Pointer(&expected.C),
		unsafe.Pointer(&callback),
	}

	if _, err := CallFunction(&cif, sym, nil, args); err != nil {
		t.Fatal(err)
	}

	if receivedArg != expected {
		t.Errorf("expected %#v, received %#v", expected, receivedArg)
	}
}

func TestCallbackStructArgWithScalar(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOARCH == "arm64" {
		t.Skip("callback struct args not supported on Windows/ARM64")
	}
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "callback_struct_and_int")
	if err != nil {
		t.Fatal(err)
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{
			types.SInt32TypeDescriptor,
			types.UInt32TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.PointerTypeDescriptor,
		}); err != nil {
		t.Fatal(err)
	}

	type Pair struct {
		A int32
		B uint32
	}

	var receivedArg1 Pair
	var receivedArg2 int64
	callback := NewCallback(func(s Pair, extra int64) {
		receivedArg1 = s
		receivedArg2 = extra
	})

	expected := Pair{10, 20}
	extra := int64(1000)
	args := []unsafe.Pointer{
		unsafe.Pointer(&expected.A),
		unsafe.Pointer(&expected.B),
		unsafe.Pointer(&extra),
		unsafe.Pointer(&callback),
	}

	if _, err := CallFunction(&cif, sym, nil, args); err != nil {
		t.Fatal(err)
	}

	if receivedArg1 != expected || receivedArg2 != extra {
		t.Errorf("expected %#v %d, received %#v %d", expected, extra, receivedArg1, receivedArg2)
	}
}

// TestStructReturn24B exercises the sret path for a struct larger than 16 bytes.
// The caller provides a buffer in rvalue, goffi points the hidden return pointer
// (RDI on System V amd64, RCX on Win64, X8 on arm64) at it, and the callee writes the struct directly
// into that buffer. This covers the >16B struct return ABI, which the other
// struct-return tests (all <= 16 bytes) do not reach.
func TestStructReturn24B(t *testing.T) {
	requireStructLib(t)

	sym, err := GetSymbol(structTestLib, "return_struct_24")
	if err != nil {
		t.Fatal(err)
	}

	tripleI64 := &types.TypeDescriptor{
		Kind:      types.StructType,
		Size:      24,
		Alignment: 8,
		Members: []*types.TypeDescriptor{
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
			types.SInt64TypeDescriptor,
		},
	}

	var cif types.CallInterface
	if err := PrepareCallInterface(&cif, types.DefaultCall, tripleI64, nil); err != nil {
		t.Fatal(err)
	}

	type TripleI64 struct{ A, B, C int64 }
	var result TripleI64
	if _, err := CallFunction(&cif, sym, unsafe.Pointer(&result), nil); err != nil {
		t.Fatal(err)
	}

	want := TripleI64{11, 22, 33}
	if result != want {
		t.Fatalf("return_struct_24() = %+v, want %+v", result, want)
	}
}
