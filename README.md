# goffi - Zero-CGO FFI for Go

[![CI](https://github.com/go-webgpu/goffi/actions/workflows/ci.yml/badge.svg)](https://github.com/go-webgpu/goffi/actions)
[![Coverage](https://img.shields.io/badge/coverage-87.1%25-brightgreen)](https://github.com/go-webgpu/goffi)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-webgpu/goffi)](https://goreportcard.com/report/github.com/go-webgpu/goffi)
[![GitHub release](https://img.shields.io/github/v/release/go-webgpu/goffi)](https://github.com/go-webgpu/goffi/releases)
[![Go version](https://img.shields.io/github/go-mod/go-version/go-webgpu/goffi)](https://github.com/go-webgpu/goffi/blob/main/go.mod)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-webgpu/goffi.svg)](https://pkg.go.dev/github.com/go-webgpu/goffi)

**Pure Go Foreign Function Interface (FFI)** for calling C libraries without CGO. Primary use case: **WebGPU bindings** for GPU computing in pure Go.

```go
// Call C functions directly from Go - no CGO required!
handle, _ := ffi.LoadLibrary("wgpu_native.dll")
wgpuCreateInstance := ffi.GetSymbol(handle, "wgpuCreateInstance")
ffi.CallFunction(&cif, wgpuCreateInstance, &result, args)
```

---

## ✨ Features

- **🚫 Zero CGO** - Pure Go, no C compiler needed
- **⚡ Fast** - ~100ns FFI overhead ([benchmarks](#performance))
- **🌐 Cross-platform** - Windows + Linux + macOS AMD64 (ARM64 planned)
- **🔒 Type-safe** - Runtime type validation with detailed errors
- **📦 Production-ready** - 87.1% test coverage, comprehensive error handling
- **🎯 WebGPU-optimized** - Designed for wgpu-native bindings

---

## 🚀 Quick Start

### Installation

```bash
go get github.com/go-webgpu/goffi
```

### Basic Example

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
	// Load standard library
	var libName, funcName string
	switch runtime.GOOS {
	case "linux":
		libName, funcName = "libc.so.6", "strlen"
	case "windows":
		libName, funcName = "msvcrt.dll", "strlen"
	default:
		panic("Unsupported OS")
	}

	handle, err := ffi.LoadLibrary(libName)
	if err != nil {
		panic(err)
	}
	defer ffi.FreeLibrary(handle)

	strlen, err := ffi.GetSymbol(handle, funcName)
	if err != nil {
		panic(err)
	}

	// Prepare call interface (reuse for multiple calls!)
	cif := &types.CallInterface{}
	err = ffi.PrepareCallInterface(
		cif,
		types.DefaultCall,                // Auto-detects platform
		types.UInt64TypeDescriptor,       // size_t return
		[]*types.TypeDescriptor{types.PointerTypeDescriptor}, // const char* arg
	)
	if err != nil {
		panic(err)
	}

	// Call strlen("Hello, goffi!")
	testStr := "Hello, goffi!\x00"
	strPtr := unsafe.Pointer(unsafe.StringData(testStr))
	var length uint64

	err = ffi.CallFunction(cif, strlen, unsafe.Pointer(&length), []unsafe.Pointer{strPtr})
	if err != nil {
		panic(err)
	}

	fmt.Printf("strlen(%q) = %d\n", testStr[:len(testStr)-1], length)
	// Output: strlen("Hello, goffi!") = 13
}
```

---

## 📊 Performance

**FFI Overhead**: ~88-114 ns/op (Windows AMD64, Intel i7-1255U)

| Benchmark | Time | vs Direct Go |
|-----------|------|--------------|
| **Empty function** | 88.09 ns | ~400x slower |
| **Integer arg** | 113.9 ns | ~500x slower |
| **String processing** | 97.81 ns | ~450x slower |

**Verdict**: ✅ **Excellent for WebGPU** (GPU calls are 1-100µs, FFI is 0.1µs = 0.1-10% overhead)

See [docs/PERFORMANCE.md](docs/PERFORMANCE.md) for comprehensive analysis, optimization strategies, and when **NOT** to use goffi.

---

## ⚠️ Known Limitations

### Critical

**Variadic functions NOT supported** (`printf`, `sprintf`, etc.)
- Workaround: Use non-variadic wrappers (`puts` instead of `printf`)
- Planned: v0.5.0 (Q3 2025)

**Struct packing** follows System V ABI only
- Windows `#pragma pack` directives NOT honored
- Workaround: Manually specify `Size`/`Alignment` in `TypeDescriptor`
- Planned: v0.2.0 (platform-specific rules)

### Architectural

- **Composite types** (structs) require manual initialization
- **Cannot interrupt** C functions mid-execution (use `CallFunctionContext` for timeouts)
- **Limited to amd64** (ARM64 planned for v0.5.0)
- **No bitfields** in structs

See [CHANGELOG.md](CHANGELOG.md#known-limitations) for full details.

---

## 📖 Documentation

- **[CHANGELOG.md](CHANGELOG.md)** - Version history, migration guides
- **[ROADMAP.md](ROADMAP.md)** - Development roadmap to v1.0
- **[docs/PERFORMANCE.md](docs/PERFORMANCE.md)** - Comprehensive performance analysis
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines
- **[SECURITY.md](SECURITY.md)** - Security policy and best practices
- **[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)** - Community standards
- **[examples/](examples/)** - Working code examples

---

## 🛠️ Advanced Usage

### Typed Error Handling

```go
import "errors"

handle, err := ffi.LoadLibrary("nonexistent.dll")
if err != nil {
	var libErr *ffi.LibraryError
	if errors.As(err, &libErr) {
		fmt.Printf("Failed to %s %q: %v\n", libErr.Operation, libErr.Name, libErr.Err)
		// Output: Failed to load "nonexistent.dll": The specified module could not be found
	}
}
```

goffi provides 5 typed error types for precise error handling:
- `InvalidCallInterfaceError` - CIF preparation failures
- `LibraryError` - Library loading/symbol lookup
- `CallingConventionError` - Unsupported calling conventions
- `TypeValidationError` - Type descriptor validation
- `UnsupportedPlatformError` - Platform not supported

### Context Support (Timeouts/Cancellation)

```go
import (
	"context"
	"time"
)

ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()

err := ffi.CallFunctionContext(ctx, cif, funcPtr, &result, args)
if err == context.DeadlineExceeded {
	fmt.Println("Function call timed out!")
}
```

### Cross-Platform Calling Conventions

```go
// Auto-detect platform (recommended)
convention := types.DefaultCall

// Or explicit:
switch runtime.GOOS {
case "windows":
	convention = types.WindowsCallingConvention // Win64 ABI
case "linux", "freebsd":
	convention = types.UnixCallingConvention   // System V AMD64
}

ffi.PrepareCallInterface(cif, convention, returnType, argTypes)
```

---

## 🏗️ Architecture

goffi uses a **4-layer architecture** for safe Go→C transitions:

```
Go Code (User Application)
    ↓ ffi.CallFunction()
runtime.cgocall (Go Runtime)
    ↓ System stack switch + GC coordination
Assembly Wrapper (Platform-specific)
    ↓ Register loading (RDI/RCX + XMM0-7)
JMP Stub (Function pointer indirection)
    ↓ Indirect jump
C Function (External Library)
```

**Key technologies**:
- `runtime.cgocall` for GC-safe stack switching
- Hand-written assembly for System V AMD64 (Linux) and Win64 (Windows) ABIs
- Runtime type validation (no codegen/reflection)

See [docs/dev/TECHNICAL_ARCHITECTURE.md](docs/dev/TECHNICAL_ARCHITECTURE.md) for deep dive (internal docs).

---

## 🗺️ Roadmap

### v0.2.0 - Usability (Q2 2025)
- **CRITICAL**: Comprehensive benchmarks vs CGO/purego ✅ **DONE!**
- Builder pattern API: `lib.Call("func").Arg(...).ReturnInt()`
- Platform-specific struct alignment (Windows `#pragma pack`)
- Type-safe argument helpers (`ffi.Int32()`, `ffi.String()`)

### v0.5.0 - Platform Expansion (Q3 2025)
- ARM64 support (Linux + macOS AAPCS64 ABI)
- **Variadic function support** (printf, sprintf, etc.)
- Callback support (C→Go calls)

### v1.0.0 - Stable Release (Q1 2026)
- API stability guarantee (SemVer 2.0)
- Security audit
- Reference implementations (WebGPU, Vulkan, SQLite bindings)
- Performance benchmarks vs CGO/purego published

See [CHANGELOG.md](CHANGELOG.md#roadmap) for detailed roadmap.

---

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...
# Current coverage: 89.1%

# Run benchmarks
go test -bench=. -benchmem ./ffi

# Platform-specific tests
go test -v ./ffi  # Auto-detects Windows/Linux
```

---

## 🌍 Platform Support

| Platform | Architecture | Status | Notes |
|----------|--------------|--------|-------|
| **Windows** | amd64 | ✅ v0.1.0 | Win64 ABI, full support |
| **Linux** | amd64 | ✅ v0.1.0 | System V ABI, full support |
| **macOS** | amd64 | ✅ v0.1.1 | System V ABI, full support |
| **FreeBSD** | amd64 | ✅ v0.1.0 | System V ABI (untested) |
| **Linux** | arm64 | 🔴 v0.5.0 | AAPCS64 ABI (planned) |
| **macOS** | arm64 | 🔴 v0.5.0 | AAPCS64 ABI (planned) |

---

## 🤝 Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Quick checklist**:
1. Fork the repository
2. Create feature branch (`git checkout -b feat/amazing-feature`)
3. Write tests (maintain 80%+ coverage)
4. Run linters (`golangci-lint run`)
5. Commit with conventional commits (`feat:`, `fix:`, `docs:`)
6. Open pull request

---

## 📜 License

MIT License - see [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

- **purego** - Inspiration for CGO-free FFI approach
- **libffi** - Reference for FFI architecture patterns
- **Go runtime** - `runtime.cgocall` for safe stack switching

---

## 🔗 Related Projects

- **[go-webgpu](https://github.com/go-webgpu/go-webgpu)** - WebGPU bindings using goffi (coming soon!)
- **[wgpu-native](https://github.com/gfx-rs/wgpu-native)** - Native WebGPU implementation

---

**Made with ❤️ for GPU computing in pure Go**

*Last updated: 2025-01-17 | goffi v0.1.0*
