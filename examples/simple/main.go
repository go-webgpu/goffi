package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
)

func main() {
	// Determine OS-specific configuration
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
		fmt.Println("Unsupported OS")
		return
	}

	// Load library
	handle, err := ffi.LoadLibrary(libName)
	if err != nil {
		fmt.Println("LoadLibrary error:", err)
		return
	}

	// Get function symbol
	sym, err := ffi.GetSymbol(handle, funcName)
	if err != nil {
		fmt.Println("GetSymbol error:", err)
		return
	}

	// Prepare call interface
	cif := &types.CallInterface{}
	rtype := types.VoidTypeDescriptor
	argtypes := []*types.TypeDescriptor{types.PointerTypeDescriptor}

	err = ffi.PrepareCallInterface(cif, convention, 1, rtype, argtypes)
	if err != nil {
		fmt.Println("PrepareCallInterface error:", err)
		return
	}

	// Prepare arguments
	str := "Hello, WebGPU!\n\x00"
	arg := unsafe.Pointer(unsafe.StringData(str))
	args := []unsafe.Pointer{arg}

	// Execute function call
	err = ffi.CallFunction(cif, sym, nil, args)
	if err != nil {
		fmt.Println("CallFunction error:", err)
	}
}
