# goffi: Pure Go FFI for WebGPU

[![Go Reference](https://pkg.go.dev/badge/github.com/go-webgpu/goffi.svg)](https://pkg.go.dev/github.com/go-webgpu/goffi)
[![CI](https://github.com/go-webgpu/goffi/actions/workflows/go.yml/badge.svg)](https://github.com/go-webgpu/goffi/actions/workflows/go.yml)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Lightweight Foreign Function Interface (FFI) for WebGPU without CGO. Provides direct GPU access in Go via wgpu-native bindings.

## Features

- ✅ **Zero C dependencies** - Pure Go implementation
- ⚡ **High performance** - Optimized assembly calls
- 🌐 **Cross-platform**:
  - Windows (amd64)
  - Linux (amd64)
- 🔧 **WebGPU integration** - Seamless work with wgpu-native
- 🔒 **Memory safe** - Strict input validation and error handling
- 📊 **Benchmark tested** - Performance-optimized critical paths

## Installation

```bash
go get github.com/go-webgpu/goffi
```

## Quick Start

### Basic Function Call

```go
package main

import (
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
        println("Unsupported OS")
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
package main

import (
    "fmt"
    "unsafe"

    "github.com/go-webgpu/goffi/ffi"
    "github.com/go-webgpu/goffi/types"
)

func main() {
    // Load WebGPU library
    wgpu := ffi.LoadLibrary("wgpu.dll")
    sym := ffi.GetSymbol(wgpu, "wgpuCreateInstance")

    // Prepare instance descriptor
    desc := types.WGPUInstanceDescriptor{
        NextInChain: nil,
    }

    // Prepare call interface
    cif := &types.CallInterface{}
    ffi.PrepareCallInterface(cif, types.WindowsCallingConvention, 
        1, types.PointerTypeDescriptor, 
        []*types.TypeDescriptor{types.PointerTypeDescriptor},
    )

    var instance types.WGPUInstance
    ffi.CallFunction(cif, sym, unsafe.Pointer(&instance), 
        []unsafe.Pointer{unsafe.Pointer(&desc)},
    )

    fmt.Println("WebGPU instance created:", instance)
}
```

## Advanced Usage

### Handling Structs

```go
// Define a WebGPU struct
vertexLayout := &types.WGPUVertexBufferLayout{
    ArrayStride: 20,
    StepMode:    types.VertexStepModeVertex,
    AttributeCount: 2,
    Attributes: []types.WGPUVertexAttribute{
        {Format: types.VertexFormatFloat32x3, Offset: 0, ShaderLocation: 0},
        {Format: types.VertexFormatFloat32x2, Offset: 12, ShaderLocation: 1},
    },
}

// Prepare call interface for struct argument
cif := &types.CallInterface{}
argTypes := []*types.TypeDescriptor{types.PointerTypeDescriptor}
ffi.PrepareCallInterface(cif, convention, 1, types.VoidTypeDescriptor, argTypes)

// Call function with struct
ffi.CallFunction(cif, createBufferFn, nil, 
    []unsafe.Pointer{unsafe.Pointer(vertexLayout)},
)
```

### Error Handling

```go
handle, err := ffi.LoadLibrary("wgpu.dll")
if err != nil {
    log.Fatalf("Failed to load library: %v", err)
}

sym, err := ffi.GetSymbol(handle, "wgpuCreateInstance")
if err != nil {
    log.Fatalf("Failed to get symbol: %v", err)
}

err = ffi.PrepareCallInterface(cif, convention, 1, rtype, argTypes)
if err != nil {
    log.Fatalf("Call interface preparation failed: %v", err)
}

err = ffi.CallFunction(cif, sym, rvalue, args)
if err != nil {
    log.Fatalf("Function call failed: %v", err)
}
```

## Performance

goffi is optimized for high-performance GPU operations:

```bash
go test -bench=. -tags "amd64 linux" ./ffi
```

Sample benchmark results:
```
BenchmarkPrepCIF-16         1,245,678      962 ns/op       0 B/op       0 allocs/op
BenchmarkCallPrintf-16         12,456     96,245 ns/op    1,024 B/op     2 allocs/op
```

## Building

### Linux
```bash
go build -tags "amd64 linux" ./...
```

### Windows
```bash
go build -tags "amd64 windows" ./...
```

## Testing

Run platform-specific tests:

```bash
# Windows tests
go test -v -cover -tags "amd64 windows" ./...

# Linux tests
go test -v -cover -tags "amd64 linux" ./...
```

## Supported Platforms

| Platform      | Architecture | Status      | Notes                     |
|---------------|--------------|-------------|---------------------------|
| Windows       | amd64        | ✅ Stable   | Full Win64 ABI support    |
| Linux         | amd64        | ✅ Stable   | Complete Unix64 support   |
| macOS         | amd64        | ⏳ Planned  | Development in progress   |
| Linux/Windows | arm64        | ⏳ Planned  | Targeting Q4 2025         |

## Architecture

```
goffi
├── ffi/             # Core FFI functionality
├── types/           # Type system and descriptors
├── internal/        # Platform-specific implementations
│   └── arch/
│       ├── amd64/   # x86-64 assembly and logic
│       └── stubs/   # Fallback implementations
├── examples/        # Usage examples
└── tests/           # Comprehensive test suite
```

## Contributing

We welcome contributions! Please see our [Contribution Guidelines](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -am 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
