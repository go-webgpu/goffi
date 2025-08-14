package ffi

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

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

	err := PrepareCallInterface(cif, convention, 1, rtype, argtypes)
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
	var libName, funcName string
	var convention types.CallingConvention
	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
		funcName = "puts"
		convention = types.UnixCallingConvention
	case "windows":
		libName = "msvcrt.dll"
		funcName = "printf"
		convention = types.WindowsCallingConvention
	default:
		t.Skip("Test requires Linux or Windows")
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

	err = PrepareCallInterface(cif, convention, 1, rtype, argtypes)
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
