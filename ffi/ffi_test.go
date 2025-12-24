package ffi

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

/*func TestFindLibrary(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Test requires Linux")
	}

	paths := []string{
		"libc.so.6",
		"/lib/x86_64-linux-gnu/libc.so.6",
		"/lib64/libc.so.6",
		"/usr/lib64/libc.so.6",
		"/usr/lib/libc.so.6",
	}

	for _, path := range paths {
		if _, err := findLibrary(path); err != nil {
			t.Logf("Library not found: %s", path)
		} else {
			t.Logf("Library found: %s", path)
			return
		}
	}
	t.Error("No standard library paths found")
}*/

func TestPrepCIF(t *testing.T) {
	cif := &types.CallInterface{}
	rtype := types.VoidTypeDescriptor
	argtypes := []*types.TypeDescriptor{types.PointerTypeDescriptor}

	var convention types.CallingConvention
	if runtime.GOOS == "windows" {
		convention = types.WindowsCallingConvention
	} else {
		convention = types.UnixCallingConvention
	}

	err := PrepareCallInterface(cif, convention, rtype, argtypes)
	if err != nil {
		t.Fatalf("PrepareCallInterface failed: %v", err)
	}
	if cif.Convention != convention ||
		cif.ArgCount != 1 ||
		cif.ReturnType != types.VoidTypeDescriptor ||
		cif.StackBytes < 8 {
		t.Errorf("Invalid CallInterface: %+v", cif)
	}
}

func TestCallPrintf(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skip("Test requires Linux or Windows")
	}

	var libName, funcName string
	var convention types.CallingConvention
	switch runtime.GOOS {
	case "linux":
		// Используем базовое имя, поиск по путям сделает findLibrary
		libName = "libc.so.6"
		funcName = "puts"
		convention = types.UnixCallingConvention
	case "windows":
		libName = "msvcrt.dll"
		funcName = "printf"
		convention = types.WindowsCallingConvention
	default:
		t.Skip("Unsupported OS")
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		t.Fatalf("LoadLibrary failed: %v", err)
	}
	sym, err := GetSymbol(handle, funcName)
	if err != nil {
		t.Fatalf("GetSymbol failed: %v", err)
	}

	cif := &types.CallInterface{}
	rtype := types.SInt32TypeDescriptor
	argtypes := []*types.TypeDescriptor{types.PointerTypeDescriptor}

	err = PrepareCallInterface(cif, convention, rtype, argtypes)
	if err != nil {
		t.Fatalf("PrepareCallInterface failed: %v", err)
	}

	str := "Hello, WebGPU!\n\x00"
	arg := unsafe.Pointer(unsafe.StringData(str))
	// IMPORTANT: avalue contains pointers TO the argument values
	// For PointerType, we pass pointer to the pointer value
	avalue := []unsafe.Pointer{unsafe.Pointer(&arg)}

	var retVal int32
	err = CallFunction(cif, sym, unsafe.Pointer(&retVal), avalue)
	if err != nil {
		t.Fatalf("CallFunction failed: %v", err)
	}

	// Check return value
	if runtime.GOOS == "windows" {
		if retVal <= 0 {
			t.Errorf("printf returned %d, expected > 0", retVal)
		}
	} else {
		if retVal < 0 {
			t.Errorf("puts returned %d, expected >= 0", retVal)
		}
	}
}

// TestPointerArgumentPassing is a regression test for GitHub Issue #4.
// It verifies that PointerType arguments are correctly dereferenced.
//
// Bug: Prior to fix, PointerType was passed as:
//
//	gpr[idx] = uintptr(avalue[idx])  // WRONG: passes address of the pointer
//
// Fixed:
//
//	gpr[idx] = *(*uintptr)(avalue[idx])  // CORRECT: dereferences to get pointer value
//
// The API contract (ffi.go line 43) specifies: []unsafe.Pointer{unsafe.Pointer(&arg)}
// This means avalue[idx] points TO the argument value, so dereference is required.
func TestPointerArgumentPassing(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skip("Test requires Linux or Windows")
	}

	// Test using strlen which takes a pointer and returns its length
	var libName, funcName string
	var convention types.CallingConvention

	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
		funcName = "strlen"
		convention = types.UnixCallingConvention
	case "windows":
		libName = "msvcrt.dll"
		funcName = "strlen"
		convention = types.WindowsCallingConvention
	default:
		t.Skip("Unsupported OS")
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		t.Fatalf("LoadLibrary failed: %v", err)
	}
	defer FreeLibrary(handle)

	sym, err := GetSymbol(handle, funcName)
	if err != nil {
		t.Fatalf("GetSymbol failed: %v", err)
	}

	cif := &types.CallInterface{}
	err = PrepareCallInterface(cif, convention, types.UInt64TypeDescriptor, []*types.TypeDescriptor{types.PointerTypeDescriptor})
	if err != nil {
		t.Fatalf("PrepareCallInterface failed: %v", err)
	}

	testCases := []struct {
		name     string
		input    string
		expected uint64
	}{
		{"empty", "\x00", 0},
		{"short", "Hello\x00", 5},
		{"longer", "Hello, World!\x00", 13},
		{"unicode", "Привет\x00", 12}, // UTF-8: 6 cyrillic chars = 12 bytes
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create pointer to string data
			ptr := unsafe.Pointer(unsafe.StringData(tc.input))

			// CRITICAL: Pass pointer TO the pointer value (documented API pattern)
			// This tests the fix for Issue #4 - PointerType dereference
			avalue := []unsafe.Pointer{unsafe.Pointer(&ptr)}

			var result uint64
			err := CallFunction(cif, sym, unsafe.Pointer(&result), avalue)
			if err != nil {
				t.Fatalf("CallFunction failed: %v", err)
			}

			if result != tc.expected {
				t.Errorf("strlen(%q) = %d, expected %d", tc.input, result, tc.expected)
			}
		})
	}
}

// TestIntegerArgumentTypes verifies all integer types are correctly handled.
// This is a regression test to ensure consistent dereference pattern across types.
func TestIntegerArgumentTypes(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skip("Test requires Linux or Windows")
	}

	// Use abs() for int32 testing
	var libName, funcName string
	var convention types.CallingConvention

	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
		funcName = "abs"
		convention = types.UnixCallingConvention
	case "windows":
		libName = "msvcrt.dll"
		funcName = "abs"
		convention = types.WindowsCallingConvention
	default:
		t.Skip("Unsupported OS")
	}

	handle, err := LoadLibrary(libName)
	if err != nil {
		t.Fatalf("LoadLibrary failed: %v", err)
	}
	defer FreeLibrary(handle)

	sym, err := GetSymbol(handle, funcName)
	if err != nil {
		t.Fatalf("GetSymbol failed: %v", err)
	}

	cif := &types.CallInterface{}
	err = PrepareCallInterface(cif, convention, types.SInt32TypeDescriptor, []*types.TypeDescriptor{types.SInt32TypeDescriptor})
	if err != nil {
		t.Fatalf("PrepareCallInterface failed: %v", err)
	}

	testCases := []struct {
		input    int32
		expected int32
	}{
		{0, 0},
		{42, 42},
		{-42, 42},
		{-2147483648, -2147483648}, // INT_MIN edge case (undefined behavior in C, but good to test)
	}

	for _, tc := range testCases {
		t.Run(
			"",
			func(t *testing.T) {
				arg := tc.input
				// CRITICAL: Pass pointer TO the value (documented API pattern)
				avalue := []unsafe.Pointer{unsafe.Pointer(&arg)}

				var result int32
				err := CallFunction(cif, sym, unsafe.Pointer(&result), avalue)
				if err != nil {
					t.Fatalf("CallFunction failed: %v", err)
				}

				// Note: abs(INT_MIN) is undefined behavior in C, skip that check
				if tc.input != -2147483648 && result != tc.expected {
					t.Errorf("abs(%d) = %d, expected %d", tc.input, result, tc.expected)
				}
			},
		)
	}
}
