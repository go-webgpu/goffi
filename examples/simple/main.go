package main

import (
	"fmt"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
)

func main() {
	var libName, funcName string
	var abi types.ABI
	switch types.RuntimeArch() {
	case "amd64":
		switch types.RuntimeOS() {
		case "linux":
			libName = "libc.so.6"
			funcName = "puts"
			abi = types.Unix64
		case "windows":
			libName = "msvcrt.dll"
			funcName = "printf"
			abi = types.Win64
		default:
			fmt.Println("Unsupported OS")
			return
		}
	default:
		fmt.Println("Unsupported architecture")
		return
	}

	handle, err := ffi.LoadLibrary(libName)
	if err != nil {
		fmt.Println("LoadLibrary:", err)
		return
	}
	sym, err := ffi.GetSymbol(handle, unsafe.Pointer(&funcName))
	if err != nil {
		fmt.Println("GetSymbol:", err)
		return
	}

	cif := &types.CIF{}
	err = ffi.PrepCIF(cif, abi, 1, types.TypeVoid, []*types.Type{types.TypePointer})
	if err != nil {
		fmt.Println("PrepCIF:", err)
		return
	}

	str := append([]byte("Hello, WebGPU!\n"), 0)
	avalue := []unsafe.Pointer{unsafe.Pointer(&str[0])}
	err = ffi.Call(cif, sym, nil, avalue)
	if err != nil {
		fmt.Println("Call:", err)
	}
}
