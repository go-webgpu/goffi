goffi - Pure Go FFI for WebGPU
================================

[![Go Reference](https://pkg.go.dev/badge/github.com/go-webgpu/goffi.svg)](https://pkg.go.dev/github.com/go-webgpu/goffi)
[![Tests](https://github.com/go-webgpu/goffi/actions/workflows/tests.yml/badge.svg)](https://github.com/go-webgpu/goffi/actions/workflows/tests.yml)

Pure Go Foreign Function Interface (FFI) for WebGPU, analogous to libffi. Provides C-API bindings for WebGPU without CGO.

## Features

- ✅ **Pure Go implementation** - no CGO dependencies
- 🚀 **High performance** - optimized assembly calls
- 🌐 **Multi-platform support**:
  - Linux (amd64)
  - Windows (amd64)
- 🔧 **WebGPU integration** - seamless work with wgpu-native

## Installation

```bash
go get github.com/go-webgpu/goffi
```

## Usage

### Basic function call

```go
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

	// Load library and function
	handle, _ := ffi.LoadLibrary(libName)
	sym, _ := ffi.GetSymbol(handle, funcName)

	// Prepare call interface
	cif := &types.CallInterface{}
	rtype := types.VoidTypeDescriptor
	argtypes := []*types.TypeDescriptor{types.PointerTypeDescriptor}
	
	ffi.PrepareCallInterface(cif, convention, 1, rtype, argtypes)

	// Prepare arguments
	str := "Hello, WebGPU!\n\x00"
	arg := unsafe.Pointer(unsafe.StringData(str))
	args := []unsafe.Pointer{arg}

	// Call function
	ffi.CallFunction(cif, sym, nil, args)
}
```

### WebGPU Integration

```go
// Create WebGPU instance
wgpu := ffi.LoadLibrary("wgpu.dll")
createInstance := ffi.GetSymbol(wgpu, "wgpuCreateInstance")

// Prepare instance descriptor
desc := types.WGPUInstanceDescriptor{
	NextInChain: nil,
}

cif := &types.CallInterface{}
ffi.PrepareCallInterface(cif, types.WindowsCallingConvention, 
	1, types.PointerTypeDescriptor, 
	[]*types.TypeDescriptor{types.PointerTypeDescriptor},
)

var instance types.WGPUInstance
ffi.CallFunction(cif, createInstance, unsafe.Pointer(&instance), 
	[]unsafe.Pointer{unsafe.Pointer(&desc)},
)

fmt.Println("WebGPU instance created:", instance)
```

## Building

### Linux
```bash
go build -tags "amd64 linux"
```

### Windows
```bash
go build -tags "amd64 windows"
```

## Testing
```bash
# Windows tests
go test -tags "amd64 windows" ./...

# Linux tests
go test -tags "amd64 linux" ./...
```

## Status

| Platform      | Architecture | Status      |
|---------------|--------------|-------------|
| Windows       | amd64        | ✅ Supported |
| Linux         | amd64        | ✅ Supported |
| macOS         | amd64        | ⏳ Planned   |
| Linux/Windows | arm64        | ⏳ Planned   |

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.
