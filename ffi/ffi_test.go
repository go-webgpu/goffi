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

// TestWindowsStackArguments verifies that functions with >4 arguments work on Windows.
// Win64 ABI: first 4 args in registers (RCX, RDX, R8, R9), args 5+ on stack.
// This is a regression test for the "stack arguments not implemented" panic.
//
// Uses CreateFileA from kernel32.dll which has 7 parameters:
//
//	HANDLE CreateFileA(
//	    LPCSTR lpFileName,                // arg1 - RCX
//	    DWORD dwDesiredAccess,            // arg2 - RDX
//	    DWORD dwShareMode,                // arg3 - R8
//	    LPSECURITY_ATTRIBUTES lpSecAttr,  // arg4 - R9
//	    DWORD dwCreationDisposition,      // arg5 - STACK
//	    DWORD dwFlagsAndAttributes,       // arg6 - STACK
//	    HANDLE hTemplateFile              // arg7 - STACK
//	)
func TestWindowsStackArguments(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Test requires Windows")
	}

	handle, err := LoadLibrary("kernel32.dll")
	if err != nil {
		t.Fatalf("LoadLibrary failed: %v", err)
	}
	defer FreeLibrary(handle)

	sym, err := GetSymbol(handle, "CreateFileA")
	if err != nil {
		t.Fatalf("GetSymbol failed: %v", err)
	}

	// Prepare CIF with 7 arguments (4 register + 3 stack)
	cif := &types.CallInterface{}
	argTypes := []*types.TypeDescriptor{
		types.PointerTypeDescriptor,  // lpFileName
		types.UInt32TypeDescriptor,   // dwDesiredAccess
		types.UInt32TypeDescriptor,   // dwShareMode
		types.PointerTypeDescriptor,  // lpSecurityAttributes
		types.UInt32TypeDescriptor,   // dwCreationDisposition
		types.UInt32TypeDescriptor,   // dwFlagsAndAttributes
		types.PointerTypeDescriptor,  // hTemplateFile
	}

	err = PrepareCallInterface(cif, types.WindowsCallingConvention, types.PointerTypeDescriptor, argTypes)
	if err != nil {
		t.Fatalf("PrepareCallInterface failed: %v", err)
	}

	// Try to open a non-existent file - should return INVALID_HANDLE_VALUE (-1)
	fileName := "C:\\__goffi_test_nonexistent_file_12345__.txt\x00"
	fileNamePtr := unsafe.Pointer(unsafe.StringData(fileName))

	// Windows constants
	const (
		GENERIC_READ          = 0x80000000
		FILE_SHARE_READ       = 0x00000001
		OPEN_EXISTING         = 3
		FILE_ATTRIBUTE_NORMAL = 0x80
		INVALID_HANDLE_VALUE  = ^uintptr(0) // -1
	)

	// Prepare arguments
	arg1 := fileNamePtr                   // lpFileName
	arg2 := uint32(GENERIC_READ)          // dwDesiredAccess
	arg3 := uint32(FILE_SHARE_READ)       // dwShareMode
	arg4 := uintptr(0)                    // lpSecurityAttributes (NULL)
	arg5 := uint32(OPEN_EXISTING)         // dwCreationDisposition (arg 5 - STACK!)
	arg6 := uint32(FILE_ATTRIBUTE_NORMAL) // dwFlagsAndAttributes (arg 6 - STACK!)
	arg7 := uintptr(0)                    // hTemplateFile (arg 7 - STACK!)

	avalue := []unsafe.Pointer{
		unsafe.Pointer(&arg1),
		unsafe.Pointer(&arg2),
		unsafe.Pointer(&arg3),
		unsafe.Pointer(&arg4),
		unsafe.Pointer(&arg5),
		unsafe.Pointer(&arg6),
		unsafe.Pointer(&arg7),
	}

	var result uintptr
	err = CallFunction(cif, sym, unsafe.Pointer(&result), avalue)
	if err != nil {
		t.Fatalf("CallFunction failed: %v", err)
	}

	// Should return INVALID_HANDLE_VALUE for non-existent file
	if result != INVALID_HANDLE_VALUE {
		t.Errorf("CreateFileA returned %v, expected INVALID_HANDLE_VALUE (%v)", result, INVALID_HANDLE_VALUE)
		t.Log("Note: If this test fails with a valid handle, the file unexpectedly exists")
	} else {
		t.Log("CreateFileA correctly returned INVALID_HANDLE_VALUE for non-existent file")
		t.Log("This confirms 7 arguments (4 register + 3 stack) are passed correctly")
	}
}

// TestWindowsStackArgumentsFileIO is a comprehensive test that creates a file,
// writes data, reads it back, and verifies correctness. This test exercises:
//   - CreateFileA: 7 arguments (4 register + 3 stack)
//   - WriteFile: 5 arguments (4 register + 1 stack)
//   - ReadFile: 5 arguments (4 register + 1 stack)
//   - CloseHandle: 1 argument
//   - DeleteFileA: 1 argument
//
// This provides strong verification that stack arguments are passed correctly.
func TestWindowsStackArgumentsFileIO(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Test requires Windows")
	}

	kernel32, err := LoadLibrary("kernel32.dll")
	if err != nil {
		t.Fatalf("LoadLibrary failed: %v", err)
	}
	defer FreeLibrary(kernel32)

	// Get all required symbols
	createFileA, err := GetSymbol(kernel32, "CreateFileA")
	if err != nil {
		t.Fatalf("GetSymbol(CreateFileA) failed: %v", err)
	}
	writeFile, err := GetSymbol(kernel32, "WriteFile")
	if err != nil {
		t.Fatalf("GetSymbol(WriteFile) failed: %v", err)
	}
	readFile, err := GetSymbol(kernel32, "ReadFile")
	if err != nil {
		t.Fatalf("GetSymbol(ReadFile) failed: %v", err)
	}
	closeHandle, err := GetSymbol(kernel32, "CloseHandle")
	if err != nil {
		t.Fatalf("GetSymbol(CloseHandle) failed: %v", err)
	}
	deleteFileA, err := GetSymbol(kernel32, "DeleteFileA")
	if err != nil {
		t.Fatalf("GetSymbol(DeleteFileA) failed: %v", err)
	}

	// Windows constants
	const (
		GENERIC_READ          = 0x80000000
		GENERIC_WRITE         = 0x40000000
		FILE_SHARE_READ       = 0x00000001
		CREATE_ALWAYS         = 2
		OPEN_EXISTING         = 3
		FILE_ATTRIBUTE_NORMAL = 0x80
		INVALID_HANDLE_VALUE  = ^uintptr(0)
	)

	// Test data - use recognizable pattern to verify correct transmission
	testData := "goffi-stack-args-test-data-12345-ABCDE"
	tempFile := "C:\\Windows\\Temp\\goffi_stack_args_test.tmp\x00"
	tempFilePtr := unsafe.Pointer(unsafe.StringData(tempFile))

	// === Step 1: Create file with CreateFileA (7 args) ===
	t.Log("Step 1: Creating file with CreateFileA (7 args: 4 register + 3 stack)")

	cifCreate := &types.CallInterface{}
	err = PrepareCallInterface(cifCreate, types.WindowsCallingConvention, types.PointerTypeDescriptor, []*types.TypeDescriptor{
		types.PointerTypeDescriptor, // lpFileName
		types.UInt32TypeDescriptor,  // dwDesiredAccess
		types.UInt32TypeDescriptor,  // dwShareMode
		types.PointerTypeDescriptor, // lpSecurityAttributes
		types.UInt32TypeDescriptor,  // dwCreationDisposition (STACK)
		types.UInt32TypeDescriptor,  // dwFlagsAndAttributes (STACK)
		types.PointerTypeDescriptor, // hTemplateFile (STACK)
	})
	if err != nil {
		t.Fatalf("PrepareCallInterface for CreateFileA failed: %v", err)
	}

	arg1 := tempFilePtr
	arg2 := uint32(GENERIC_READ | GENERIC_WRITE)
	arg3 := uint32(0)
	arg4 := uintptr(0)
	arg5 := uint32(CREATE_ALWAYS)
	arg6 := uint32(FILE_ATTRIBUTE_NORMAL)
	arg7 := uintptr(0)

	var fileHandle uintptr
	err = CallFunction(cifCreate, createFileA, unsafe.Pointer(&fileHandle), []unsafe.Pointer{
		unsafe.Pointer(&arg1),
		unsafe.Pointer(&arg2),
		unsafe.Pointer(&arg3),
		unsafe.Pointer(&arg4),
		unsafe.Pointer(&arg5),
		unsafe.Pointer(&arg6),
		unsafe.Pointer(&arg7),
	})
	if err != nil {
		t.Fatalf("CreateFileA call failed: %v", err)
	}

	if fileHandle == INVALID_HANDLE_VALUE {
		t.Fatal("CreateFileA returned INVALID_HANDLE_VALUE - cannot create test file")
	}
	t.Logf("  CreateFileA succeeded, handle: %v", fileHandle)

	// === Step 2: Write data with WriteFile (5 args) ===
	t.Log("Step 2: Writing data with WriteFile (5 args: 4 register + 1 stack)")

	cifWrite := &types.CallInterface{}
	err = PrepareCallInterface(cifWrite, types.WindowsCallingConvention, types.SInt32TypeDescriptor, []*types.TypeDescriptor{
		types.PointerTypeDescriptor, // hFile
		types.PointerTypeDescriptor, // lpBuffer
		types.UInt32TypeDescriptor,  // nNumberOfBytesToWrite
		types.PointerTypeDescriptor, // lpNumberOfBytesWritten
		types.PointerTypeDescriptor, // lpOverlapped (STACK!)
	})
	if err != nil {
		t.Fatalf("PrepareCallInterface for WriteFile failed: %v", err)
	}

	dataBytes := []byte(testData)
	var bytesWritten uint32
	wArg1 := fileHandle
	wArg2 := unsafe.Pointer(&dataBytes[0])
	wArg3 := uint32(len(dataBytes))
	wArg4 := unsafe.Pointer(&bytesWritten)
	wArg5 := uintptr(0) // lpOverlapped - STACK ARGUMENT!

	var writeResult int32
	err = CallFunction(cifWrite, writeFile, unsafe.Pointer(&writeResult), []unsafe.Pointer{
		unsafe.Pointer(&wArg1),
		unsafe.Pointer(&wArg2),
		unsafe.Pointer(&wArg3),
		unsafe.Pointer(&wArg4),
		unsafe.Pointer(&wArg5),
	})
	if err != nil {
		t.Fatalf("WriteFile call failed: %v", err)
	}

	if writeResult == 0 {
		t.Fatal("WriteFile returned FALSE - write failed")
	}
	if bytesWritten != uint32(len(dataBytes)) {
		t.Fatalf("WriteFile wrote %d bytes, expected %d", bytesWritten, len(dataBytes))
	}
	t.Logf("  WriteFile succeeded, wrote %d bytes", bytesWritten)

	// === Step 3: Close file ===
	t.Log("Step 3: Closing file with CloseHandle")

	cifClose := &types.CallInterface{}
	err = PrepareCallInterface(cifClose, types.WindowsCallingConvention, types.SInt32TypeDescriptor, []*types.TypeDescriptor{
		types.PointerTypeDescriptor,
	})
	if err != nil {
		t.Fatalf("PrepareCallInterface for CloseHandle failed: %v", err)
	}

	cArg1 := fileHandle
	var closeResult int32
	err = CallFunction(cifClose, closeHandle, unsafe.Pointer(&closeResult), []unsafe.Pointer{
		unsafe.Pointer(&cArg1),
	})
	if err != nil {
		t.Fatalf("CloseHandle call failed: %v", err)
	}
	t.Log("  CloseHandle succeeded")

	// === Step 4: Reopen and read file (5 args) ===
	t.Log("Step 4: Reopening file with CreateFileA (7 args)")

	arg2 = uint32(GENERIC_READ)
	arg5 = uint32(OPEN_EXISTING)
	err = CallFunction(cifCreate, createFileA, unsafe.Pointer(&fileHandle), []unsafe.Pointer{
		unsafe.Pointer(&arg1),
		unsafe.Pointer(&arg2),
		unsafe.Pointer(&arg3),
		unsafe.Pointer(&arg4),
		unsafe.Pointer(&arg5),
		unsafe.Pointer(&arg6),
		unsafe.Pointer(&arg7),
	})
	if err != nil {
		t.Fatalf("CreateFileA (reopen) call failed: %v", err)
	}
	if fileHandle == INVALID_HANDLE_VALUE {
		t.Fatal("CreateFileA (reopen) returned INVALID_HANDLE_VALUE")
	}
	t.Logf("  CreateFileA (reopen) succeeded, handle: %v", fileHandle)

	// Read data back
	t.Log("Step 5: Reading data with ReadFile (5 args: 4 register + 1 stack)")

	cifRead := &types.CallInterface{}
	err = PrepareCallInterface(cifRead, types.WindowsCallingConvention, types.SInt32TypeDescriptor, []*types.TypeDescriptor{
		types.PointerTypeDescriptor, // hFile
		types.PointerTypeDescriptor, // lpBuffer
		types.UInt32TypeDescriptor,  // nNumberOfBytesToRead
		types.PointerTypeDescriptor, // lpNumberOfBytesRead
		types.PointerTypeDescriptor, // lpOverlapped (STACK!)
	})
	if err != nil {
		t.Fatalf("PrepareCallInterface for ReadFile failed: %v", err)
	}

	readBuffer := make([]byte, len(dataBytes)+10)
	var bytesRead uint32
	rArg1 := fileHandle
	rArg2 := unsafe.Pointer(&readBuffer[0])
	rArg3 := uint32(len(readBuffer))
	rArg4 := unsafe.Pointer(&bytesRead)
	rArg5 := uintptr(0) // lpOverlapped - STACK ARGUMENT!

	var readResult int32
	err = CallFunction(cifRead, readFile, unsafe.Pointer(&readResult), []unsafe.Pointer{
		unsafe.Pointer(&rArg1),
		unsafe.Pointer(&rArg2),
		unsafe.Pointer(&rArg3),
		unsafe.Pointer(&rArg4),
		unsafe.Pointer(&rArg5),
	})
	if err != nil {
		t.Fatalf("ReadFile call failed: %v", err)
	}

	if readResult == 0 {
		t.Fatal("ReadFile returned FALSE - read failed")
	}
	t.Logf("  ReadFile succeeded, read %d bytes", bytesRead)

	// === Step 6: Verify data ===
	t.Log("Step 6: Verifying data integrity")

	readData := string(readBuffer[:bytesRead])
	if readData != testData {
		t.Fatalf("Data mismatch!\n  Written: %q\n  Read:    %q", testData, readData)
	}
	t.Logf("  Data verified: %q", readData)

	// === Step 7: Cleanup ===
	t.Log("Step 7: Cleanup")

	cArg1 = fileHandle
	err = CallFunction(cifClose, closeHandle, unsafe.Pointer(&closeResult), []unsafe.Pointer{
		unsafe.Pointer(&cArg1),
	})
	if err != nil {
		t.Logf("  CloseHandle (cleanup) failed: %v", err)
	}

	cifDelete := &types.CallInterface{}
	err = PrepareCallInterface(cifDelete, types.WindowsCallingConvention, types.SInt32TypeDescriptor, []*types.TypeDescriptor{
		types.PointerTypeDescriptor,
	})
	if err == nil {
		dArg1 := tempFilePtr
		var deleteResult int32
		_ = CallFunction(cifDelete, deleteFileA, unsafe.Pointer(&deleteResult), []unsafe.Pointer{
			unsafe.Pointer(&dArg1),
		})
	}

	t.Log("=== SUCCESS ===")
	t.Log("All stack argument tests passed:")
	t.Log("  - CreateFileA (7 args: 4 reg + 3 stack)")
	t.Log("  - WriteFile (5 args: 4 reg + 1 stack)")
	t.Log("  - ReadFile (5 args: 4 reg + 1 stack)")
	t.Log("  - Data integrity verified: written == read")
}
