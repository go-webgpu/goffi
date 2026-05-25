// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build (linux || darwin || freebsd) && (amd64 || arm64)

// Command variadic-test compiles the bundled C test library and exercises
// PrepareVariadicCallInterface on the current platform.
//
// On Apple ARM64 the key invariant under test is that variadic arguments are
// passed on the stack (not in registers), per Apple's AAPCS64 extension.
// On all other platforms the call is functionally identical to a normal
// PrepareCallInterface call.
//
// Usage:
//
//	go run github.com/go-webgpu/goffi/cmd/variadic-test
//
// Exit code 0 = PASS, 1 = FAIL.
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

func main() {
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	soPath, err := buildLib()
	if err != nil {
		fmt.Fprintf(os.Stderr, "SKIP: failed to compile test library (gcc required): %v\n", err)
		// Exit 0 so CI passes on machines without gcc.
		os.Exit(0)
	}

	lib, err := ffi.LoadLibrary(soPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: LoadLibrary(%q): %v\n", soPath, err)
		os.Exit(1)
	}
	defer func() { _ = ffi.FreeLibrary(lib) }()

	pass := true
	pass = testSumVariadic(lib) && pass
	pass = testTwoFixed(lib) && pass

	if pass {
		fmt.Println("PASS — variadic functions work on this platform")
	} else {
		fmt.Println("FAIL — one or more variadic tests failed")
		os.Exit(1)
	}
}

// buildLib compiles testdata/structtest.c into a shared library and returns
// its absolute path.  The source file lives next to the goffi source tree,
// which cmd/variadic-test references via a relative path that stays inside
// the module root.
func buildLib() (string, error) {
	// Resolve testdata relative to this source file's directory at runtime
	// so the command works regardless of cwd.
	_, thisFile, _, _ := runtime.Caller(0)
	srcDir := filepath.Dir(thisFile)
	// srcDir is …/cmd/variadic-test; go up two levels to reach the module root.
	moduleRoot := filepath.Join(srcDir, "..", "..")
	srcPath := filepath.Join(moduleRoot, "ffi", "testdata", "structtest.c")

	var soPath string
	switch runtime.GOOS {
	case "darwin":
		soPath = filepath.Join(os.TempDir(), "libstructtest.dylib")
	default:
		soPath = filepath.Join(os.TempDir(), "libstructtest.so")
	}

	cc := os.Getenv("CC")
	if cc == "" {
		cc = "gcc"
	}

	cmd := exec.Command(cc, "-shared", "-fPIC", "-O2", "-o", soPath, srcPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", cc, err)
	}
	return soPath, nil
}

// testSumVariadic tests sum_variadic(int64_t count, ...) with count=3 and
// three variadic int64_t arguments: 10, 20, 30.  Expected return: 60.
func testSumVariadic(lib unsafe.Pointer) bool {
	sym, err := ffi.GetSymbol(lib, "sum_variadic")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: GetSymbol(sum_variadic): %v\n", err)
		return false
	}

	allArgs := []*types.TypeDescriptor{
		types.SInt64TypeDescriptor, // count (fixed)
		types.SInt64TypeDescriptor, // arg1 (variadic)
		types.SInt64TypeDescriptor, // arg2 (variadic)
		types.SInt64TypeDescriptor, // arg3 (variadic)
	}

	var cif types.CallInterface
	if err := ffi.PrepareVariadicCallInterface(
		&cif,
		types.DefaultCall,
		1, // nfixedargs: only 'count' is fixed
		types.SInt64TypeDescriptor,
		allArgs,
	); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: PrepareVariadicCallInterface(sum_variadic): %v\n", err)
		return false
	}

	count := int64(3)
	a1 := int64(10)
	a2 := int64(20)
	a3 := int64(30)

	avalue := []unsafe.Pointer{
		unsafe.Pointer(&count),
		unsafe.Pointer(&a1),
		unsafe.Pointer(&a2),
		unsafe.Pointer(&a3),
	}

	var result int64
	if err := ffi.CallFunction(&cif, sym, unsafe.Pointer(&result), avalue); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: CallFunction(sum_variadic): %v\n", err)
		return false
	}

	const want = int64(60)
	if result != want {
		fmt.Fprintf(os.Stderr, "FAIL: sum_variadic(3, 10, 20, 30) = %d, want %d\n", result, want)
		return false
	}

	fmt.Printf("  sum_variadic(3, 10, 20, 30) = %d (want %d) OK\n", result, want)
	return true
}

// testTwoFixed tests variadic_two_fixed(int64_t a, int64_t b, ...) with
// a=100, b=200, and one variadic int64_t: 300.  Expected return: 600.
func testTwoFixed(lib unsafe.Pointer) bool {
	sym, err := ffi.GetSymbol(lib, "variadic_two_fixed")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: GetSymbol(variadic_two_fixed): %v\n", err)
		return false
	}

	allArgs := []*types.TypeDescriptor{
		types.SInt64TypeDescriptor, // a (fixed)
		types.SInt64TypeDescriptor, // b (fixed)
		types.SInt64TypeDescriptor, // extra (variadic)
	}

	var cif types.CallInterface
	if err := ffi.PrepareVariadicCallInterface(
		&cif,
		types.DefaultCall,
		2, // nfixedargs: a and b are fixed
		types.SInt64TypeDescriptor,
		allArgs,
	); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: PrepareVariadicCallInterface(variadic_two_fixed): %v\n", err)
		return false
	}

	a := int64(100)
	b := int64(200)
	extra := int64(300)

	avalue := []unsafe.Pointer{
		unsafe.Pointer(&a),
		unsafe.Pointer(&b),
		unsafe.Pointer(&extra),
	}

	var result int64
	if err := ffi.CallFunction(&cif, sym, unsafe.Pointer(&result), avalue); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: CallFunction(variadic_two_fixed): %v\n", err)
		return false
	}

	const want = int64(600)
	if result != want {
		fmt.Fprintf(os.Stderr, "FAIL: variadic_two_fixed(100, 200, 300) = %d, want %d\n", result, want)
		return false
	}

	fmt.Printf("  variadic_two_fixed(100, 200, 300) = %d (want %d) OK\n", result, want)
	return true
}
