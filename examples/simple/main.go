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

	switch runtime.GOOS {
	case "linux":
		libName = "libc.so.6"
		funcName = "puts"
	case "windows":
		libName = "msvcrt.dll"
		funcName = "printf"
	case "darwin":
		libName = "libSystem.B.dylib"
		funcName = "puts"
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
	defer ffi.FreeLibrary(handle) // Clean up when done

	// Get function symbol
	sym, err := ffi.GetSymbol(handle, funcName)
	if err != nil {
		fmt.Println("GetSymbol error:", err)
		return
	}

	// Prepare call interface
	cif := &types.CallInterface{}
	var rtype *types.TypeDescriptor
	if runtime.GOOS == "darwin" {
		rtype = types.SInt32TypeDescriptor
	} else {
		rtype = types.VoidTypeDescriptor
	}
	argtypes := []*types.TypeDescriptor{types.PointerTypeDescriptor}

	err = ffi.PrepareCallInterface(cif, types.DefaultCall, rtype, argtypes)
	if err != nil {
		fmt.Println("PrepareCallInterface error:", err)
		return
	}

	// Prepare arguments
	str := "Hello, WebGPU!\n\x00"
	if runtime.GOOS == "darwin" {
		cstr := unsafe.Pointer(unsafe.StringData(str))

		// IMPORTANT: args contains pointers to argument *storage*
		args := []unsafe.Pointer{unsafe.Pointer(&cstr)}

		var ret int32
		if err := ffi.CallFunction(cif, sym, unsafe.Pointer(&ret), args); err != nil {
			fmt.Println("CallFunction error:", err)
			return
		}

	} else {
		arg := unsafe.Pointer(unsafe.StringData(str))
		args := []unsafe.Pointer{arg}

		// Execute function call
		err = ffi.CallFunction(cif, sym, nil, args)
		if err != nil {
			fmt.Println("CallFunction error:", err)
		}
	}
}
