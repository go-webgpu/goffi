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
	avalue := []unsafe.Pointer{arg}

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
