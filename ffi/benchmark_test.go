package ffi

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// BenchmarkGoffiOverhead measures the overhead of goffi FFI call with empty function.
// This establishes the baseline cost of runtime.cgocall + assembly wrapper.
func BenchmarkGoffiOverhead(b *testing.B) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		b.Skip("Benchmark requires Linux, Windows, or MacOS")
	}

	var libName, funcName string
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
		funcName = "getpid" // Simple syscall wrapper, negligible C cost
	case "windows":
		libName = "kernel32.dll"
		funcName = "GetCurrentProcessId"
	case "darwin":
		libName = "libSystem.B.dylib"
		funcName = "getpid" // Simple syscall wrapper, negligible C cost
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		b.Fatalf("LoadLibrary failed: %v", err)
	}
	defer FreeLibrary(handle)

	sym, err := GetSymbol(handle, funcName)
	if err != nil {
		b.Fatalf("GetSymbol failed: %v", err)
	}

	cif := &types.CallInterface{}
	err = PrepareCallInterface(cif, types.DefaultCall, types.SInt32TypeDescriptor, nil)
	if err != nil {
		b.Fatalf("PrepareCallInterface failed: %v", err)
	}

	var result int32

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CallFunction(cif, sym, unsafe.Pointer(&result), nil)
	}
}

// BenchmarkGoffiIntArgs measures performance with integer arguments (GP registers).
func BenchmarkGoffiIntArgs(b *testing.B) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		b.Skip("Benchmark requires Linux, Windows, or macOS")
	}

	var libName, funcName string
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
		funcName = "abs" // int abs(int x)
	case "darwin":
		libName = "libSystem.B.dylib"
		funcName = "abs"
	case "windows":
		libName = "msvcrt.dll"
		funcName = "abs"
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		b.Fatalf("LoadLibrary failed: %v", err)
	}
	defer FreeLibrary(handle)

	sym, err := GetSymbol(handle, funcName)
	if err != nil {
		b.Fatalf("GetSymbol failed: %v", err)
	}

	cif := &types.CallInterface{}
	err = PrepareCallInterface(cif, types.DefaultCall,
		types.SInt32TypeDescriptor,
		[]*types.TypeDescriptor{types.SInt32TypeDescriptor})
	if err != nil {
		b.Fatalf("PrepareCallInterface failed: %v", err)
	}

	var result int32
	arg := int32(-42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CallFunction(cif, sym, unsafe.Pointer(&result), []unsafe.Pointer{
			unsafe.Pointer(&arg),
		})
	}
}

// BenchmarkGoffiStringOutput measures performance with string output (common case).
func BenchmarkGoffiStringOutput(b *testing.B) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		b.Skip("Benchmark requires Linux, Windows, or macOS")
	}

	var libName, funcName string
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
		funcName = "strlen" // size_t strlen(const char* s)
	case "darwin":
		libName = "libSystem.B.dylib"
		funcName = "strlen"
	case "windows":
		libName = "msvcrt.dll"
		funcName = "strlen"
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		b.Fatalf("LoadLibrary failed: %v", err)
	}
	defer FreeLibrary(handle)

	sym, err := GetSymbol(handle, funcName)
	if err != nil {
		b.Fatalf("GetSymbol failed: %v", err)
	}

	cif := &types.CallInterface{}
	err = PrepareCallInterface(cif, types.DefaultCall,
		types.UInt64TypeDescriptor,
		[]*types.TypeDescriptor{types.PointerTypeDescriptor})
	if err != nil {
		b.Fatalf("PrepareCallInterface failed: %v", err)
	}

	testStr := "Hello, WebGPU FFI!\x00"
	strPtr := unsafe.Pointer(unsafe.StringData(testStr))
	var result uint64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CallFunction(cif, sym, unsafe.Pointer(&result), []unsafe.Pointer{unsafe.Pointer(&strPtr)})
	}
}

// BenchmarkGoffiMultipleArgs measures performance with multiple arguments.
func BenchmarkGoffiMultipleArgs(b *testing.B) {
	// Note this segfaults
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		b.Skip("Benchmark requires Linux or macOS (libm)")
	}

	var libName string
	switch runtime.GOOS {
	case "linux":
		libName = "libm.so.6"
	case "darwin":
		libName = "libm.dylib"
	}

	// pow(double x, double y) - 2 double args, 1 double return
	handle, err := LoadLibrary(libName)
	if err != nil {
		b.Skipf("LoadLibrary(%s) failed: %v", libName, err)
		return
	}
	defer FreeLibrary(handle)

	sym, err := GetSymbol(handle, "pow")
	if err != nil {
		b.Skipf("GetSymbol(pow) failed: %v", err)
		return
	}

	cif := &types.CallInterface{}
	err = PrepareCallInterface(cif, types.DefaultCall,
		types.DoubleTypeDescriptor,
		[]*types.TypeDescriptor{
			types.DoubleTypeDescriptor,
			types.DoubleTypeDescriptor,
		})
	if err != nil {
		b.Fatalf("PrepareCallInterface failed: %v", err)
	}

	arg1 := 2.0
	arg2 := 3.0
	var result float64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CallFunction(cif, sym, unsafe.Pointer(&result), []unsafe.Pointer{
			unsafe.Pointer(&arg1),
			unsafe.Pointer(&arg2),
		})
	}
}

// BenchmarkDirectGo provides a baseline - direct Go function call.
func BenchmarkDirectGo(b *testing.B) {
	// Baseline: pure Go function call
	fn := func(x int32) int32 {
		if x < 0 {
			return -x
		}
		return x
	}

	arg := int32(-42)
	var result int32

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result = fn(arg)
	}
	_ = result
}

// BenchmarkPrepareCallInterface measures CIF preparation overhead (one-time cost).
func BenchmarkPrepareCallInterface(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cif := &types.CallInterface{}
		_ = PrepareCallInterface(cif, types.DefaultCall,
			types.SInt32TypeDescriptor,
			[]*types.TypeDescriptor{
				types.SInt32TypeDescriptor,
				types.DoubleTypeDescriptor,
				types.PointerTypeDescriptor,
			})
	}
}

// BenchmarkLoadLibrary measures dynamic library loading overhead.
func BenchmarkLoadLibrary(b *testing.B) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		b.Skip("Benchmark requires Linux, Windows, or macOS")
	}

	var libName string
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
	case "darwin":
		libName = "libSystem.B.dylib"
	case "windows":
		libName = "kernel32.dll"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handle, err := LoadLibrary(libName)
		if err != nil {
			b.Fatalf("LoadLibrary failed: %v", err)
		}
		_ = FreeLibrary(handle)
	}
}

// BenchmarkGetSymbol measures symbol lookup overhead.
func BenchmarkGetSymbol(b *testing.B) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		b.Skip("Benchmark requires Linux, Windows, or macOS")
	}

	var libName, funcName string
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
		funcName = "strlen"
	case "darwin":
		libName = "libSystem.B.dylib"
		funcName = "getpid"
	case "windows":
		libName = "kernel32.dll"
		funcName = "GetCurrentProcessId"
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		b.Fatalf("LoadLibrary failed: %v", err)
	}
	defer FreeLibrary(handle)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetSymbol(handle, funcName)
	}
}

// Benchmark comparison matrix - for docs/PERFORMANCE.md
//
// Expected results (approximate):
// BenchmarkGoffiOverhead:          ~200-250 ns/op
// BenchmarkGoffiIntArgs:           ~220-270 ns/op
// BenchmarkGoffiStringOutput:      ~230-280 ns/op
// BenchmarkGoffiMultipleArgs:      ~240-290 ns/op
// BenchmarkDirectGo:               ~2-5 ns/op (baseline)
// BenchmarkPrepareCallInterface:   ~500-1000 ns/op (one-time)
// BenchmarkLoadLibrary:            ~50-100 µs/op (one-time)
// BenchmarkGetSymbol:              ~100-500 ns/op (one-time)
//
// Key insights:
// - FFI overhead: ~230ns (acceptable for WebGPU - calls are rare)
// - NOT acceptable for: tight loops, hot-path math
// - One-time costs (LoadLibrary, PrepareCallInterface) amortize over many calls
