# goffi — Zero-CGO FFI for Go

[![CI](https://github.com/go-webgpu/goffi/actions/workflows/ci.yml/badge.svg)](https://github.com/go-webgpu/goffi/actions)
[![codecov](https://codecov.io/gh/go-webgpu/goffi/graph/badge.svg)](https://codecov.io/gh/go-webgpu/goffi)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-webgpu/goffi)](https://goreportcard.com/report/github.com/go-webgpu/goffi)
[![GitHub release](https://img.shields.io/github/v/release/go-webgpu/goffi)](https://github.com/go-webgpu/goffi/releases)
[![Go version](https://img.shields.io/github/go-mod-go-version/go-webgpu/goffi)](https://github.com/go-webgpu/goffi/blob/main/go.mod)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-webgpu/goffi.svg)](https://pkg.go.dev/github.com/go-webgpu/goffi)
[![Dev.to](https://img.shields.io/badge/dev.to-deep%20dive-0A0A0A?logo=devdotto)](https://dev.to/kolkov/goffi-zero-cgo-foreign-function-interface-for-go-how-we-call-c-libraries-without-a-c-compiler-ca5)

**Pure Go Foreign Function Interface** for calling C libraries without CGO.
Designed for WebGPU and GPU computing — zero C dependencies, zero per-call allocations, 88–114 ns overhead.

> **Deep dive:** [How We Call C Libraries Without a C Compiler](https://dev.to/kolkov/goffi-zero-cgo-foreign-function-interface-for-go-how-we-call-c-libraries-without-a-c-compiler-ca5) — architecture, assembly, callbacks, and ecosystem.

```go
// Load library, prepare once, call many times — no CGO required
handle, _ := ffi.LoadLibrary("wgpu_native.dll")
sym, _ := ffi.GetSymbol(handle, "wgpuCreateInstance")

cif := &types.CallInterface{}
ffi.PrepareCallInterface(cif, types.DefaultCall, returnType, argTypes)
ffi.CallFunction(cif, sym, unsafe.Pointer(&result), args)
```

---

## Features

| | Feature | Details |
|---|---------|---------|
| **Zero CGO** | Pure Go | No C compiler needed. `go get` and build. |
| **Fast** | 88–114 ns/op | Pre-computed CIF, zero per-call allocations |
| **Cross-platform** | 7 targets | Windows, Linux, macOS, FreeBSD × AMD64 + ARM64 |
| **Callbacks** | C→Go safe | `crosscall2` integration, works from any C thread |
| **Type-safe** | Runtime validation | 5 typed error types with `errors.As()` support |
| **Struct passing** | Full ABI | ≤8B (RAX), 9–16B (RAX+RDX), >16B (sret) |
| **Context** | Timeouts | `CallFunctionContext(ctx, ...)` cancellation |
| **Tested** | 89% coverage | CI on Linux, Windows, macOS |

---

## Quick Start

### Installation

```bash
go get github.com/go-webgpu/goffi
```

### Requirements

goffi requires `CGO_ENABLED=0`. This is automatic when no C compiler is installed or when cross-compiling. If you have gcc/clang:

```bash
CGO_ENABLED=0 go build ./...
```

> **Why?** goffi uses Go's `cgo_import_dynamic` for dynamic library loading, which only activates when CGO is disabled.

### Example: Calling strlen

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
	// Load platform-specific C library
	libName := "libc.so.6"
	if runtime.GOOS == "windows" {
		libName = "msvcrt.dll"
	}

	handle, err := ffi.LoadLibrary(libName)
	if err != nil {
		panic(err)
	}
	defer ffi.FreeLibrary(handle)

	strlen, err := ffi.GetSymbol(handle, "strlen")
	if err != nil {
		panic(err)
	}

	// Prepare call interface once — reuse for all subsequent calls
	cif := &types.CallInterface{}
	err = ffi.PrepareCallInterface(
		cif,
		types.DefaultCall,                                     // auto-detects platform ABI
		types.UInt64TypeDescriptor,                            // return: size_t
		[]*types.TypeDescriptor{types.PointerTypeDescriptor},  // arg: const char*
	)
	if err != nil {
		panic(err)
	}

	// Call strlen — avalue elements are pointers TO argument values
	testStr := "Hello, goffi!\x00"
	strPtr := uintptr(unsafe.Pointer(unsafe.StringData(testStr)))
	var length uint64

	err = ffi.CallFunction(cif, strlen, unsafe.Pointer(&length), []unsafe.Pointer{unsafe.Pointer(&strPtr)})
	if err != nil {
		panic(err)
	}

	fmt.Printf("strlen(%q) = %d\n", testStr[:len(testStr)-1], length)
	// Output: strlen("Hello, goffi!") = 13
}
```

---

## Performance

**FFI overhead: 88–114 ns/op** (Windows AMD64, Intel i7-1255U)

| Benchmark | Time | Allocations |
|-----------|------|-------------|
| Empty function (`getpid`) | 88 ns | 2 allocs |
| Integer argument (`abs`) | 114 ns | 3 allocs |
| String processing (`strlen`) | 98 ns | 3 allocs |

At 60 FPS with ~50 FFI calls per frame, overhead is **5 µs per frame** — 0.03% of the 16.6 ms budget. Unmeasurable in profiling.

See [docs/PERFORMANCE.md](docs/PERFORMANCE.md) for detailed analysis, optimization strategies, and when NOT to use goffi.

---

## Architecture

goffi transitions from Go's managed runtime to C code through three layers:

```
Go Code
  │  ffi.CallFunction()
  ▼
runtime.cgocall               ← Go runtime: system stack switch, GC coordination
  │
  ▼
Assembly Wrapper              ← Hand-written: load GP/SSE registers per ABI
  │  CALL target_function
  ▼
C Function                    ← External library
```

**Three ABIs, hand-written assembly for each:**

| ABI | GP Registers | FP Registers | Notes |
|-----|-------------|-------------|-------|
| System V AMD64 | RDI, RSI, RDX, RCX, R8, R9 | XMM0–XMM7 | Linux, macOS, FreeBSD |
| Win64 | RCX, RDX, R8, R9 | XMM0–XMM3 | 32-byte shadow space mandatory |
| AAPCS64 | X0–X7 | D0–D7 | HFA support for ARM64 |

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full technical deep dive.

---

## Callbacks (C → Go)

WebGPU fires async callbacks from internal Metal/Vulkan threads. These threads have no goroutine — calling Go directly would crash.

goffi uses `crosscall2` for safe C→Go transitions from any thread:

```go
cb := ffi.NewCallback(func(status uint32, adapter uintptr, msg uintptr, ud uintptr) {
    // Safe even when called from a C thread
    result.handle = adapter
    close(done)
})

ffi.CallFunction(cif, wgpuRequestAdapter, nil, args)
<-done // Wait for GPU driver callback
```

2000 pre-compiled trampoline entries per process. AMD64: 5 bytes/entry. ARM64: 8 bytes/entry.

---

## Error Handling

Five typed error types for precise diagnostics:

```go
handle, err := ffi.LoadLibrary("nonexistent.dll")
if err != nil {
	var libErr *ffi.LibraryError
	if errors.As(err, &libErr) {
		fmt.Printf("Failed to %s %q: %v\n", libErr.Operation, libErr.Name, libErr.Err)
	}
}
```

| Error Type | When |
|------------|------|
| `InvalidCallInterfaceError` | CIF preparation failures |
| `LibraryError` | Library loading / symbol lookup |
| `CallingConventionError` | Unsupported calling convention |
| `TypeValidationError` | Invalid type descriptor |
| `UnsupportedPlatformError` | Platform not supported |

---

## Comparison: goffi vs purego vs CGO

| Feature | **goffi** | purego | CGO |
|---------|-----------|--------|-----|
| C compiler required | No | No | Yes |
| API style | libffi-like (prepare once, call many) | reflect-based (RegisterFunc) | Native |
| Per-call allocations | Zero (CIF reusable) | reflect + sync.Pool per call | Zero |
| Struct pass/return | Full (RAX+RDX, sret) | Partial (no Windows structs) | Full |
| Callback float returns | XMM0 in asm | Not supported (panic) | Full |
| ARM64 HFA detection | Recursive (nested structs) | Partial (bug in nested path) | Full |
| Typed errors | 5 types + errors.As() | Generic | N/A |
| Context support | Timeouts/cancellation | No | No |
| C-thread callbacks | crosscall2 | crosscall2 | Full |
| String/bool/slice args | Raw pointers only | Auto-marshaling | Full |
| Platform breadth | 7 targets | 8 GOARCH / 20+ OS×ARCH | All |
| AMD64 overhead | 88–114 ns | Not published | ~140 ns (Go 1.26 claims ~30% reduction) |

**Choose goffi** for GPU/real-time workloads: struct passing, zero per-call overhead, callback float returns, typed errors.

**Choose purego** for general-purpose bindings: string auto-marshaling, broad architecture support, less boilerplate.

**See also:** [JupiterRider/ffi](https://github.com/JupiterRider/ffi) — pure Go binding for libffi via purego. Supports struct pass/return and variadic functions; requires libffi at runtime.

---

## Known Limitations

**Windows: C++ exceptions may crash the program** ([#12516](https://github.com/golang/go/issues/12516))
- Go runtime limitation, not goffi-specific. Go 1.22+ added partial SEH support ([#58542](https://github.com/golang/go/issues/58542)), but edge cases remain.
- Workaround: build native libraries with `panic=abort`.

**Windows: float return values not captured from XMM0**
- `syscall.SyscallN` returns RAX only. Go `syscall` package limitation.

**Variadic functions not supported** (`printf`, `sprintf`)
- Use non-variadic wrappers. Planned for v0.5.0.

**Struct packing follows System V ABI only**
- Windows `#pragma pack` not honored. Manually specify `Size`/`Alignment` in `TypeDescriptor`.

**No bitfields** in struct types.

**Unix: duplicate symbol conflict with purego** ([#22](https://github.com/go-webgpu/goffi/issues/22))
- When using goffi and purego in the same binary with `CGO_ENABLED=0`, the linker reports `duplicated definition of symbol _cgo_init`. Both libraries include `internal/fakecgo` which defines identical runtime symbols.
- Workaround: build with `-tags nofakecgo` to disable goffi's fakecgo, relying on purego's copy:
  ```bash
  CGO_ENABLED=0 go build -tags nofakecgo ./...
  ```

---

## Platform Support

| Platform | Arch | ABI | Since | CI |
|----------|------|-----|-------|----|
| Windows | amd64 | Win64 | v0.1.0 | Tested |
| Windows | arm64 | AAPCS64 | v0.5.0 | Tested (Snapdragon X) |
| Linux | amd64 | System V | v0.1.0 | Tested |
| Linux | arm64 | AAPCS64 | v0.3.0 | Cross-compile verified |
| macOS | amd64 | System V | v0.1.1 | Tested |
| macOS | arm64 | AAPCS64 | v0.3.7 | Tested (M3 Pro) |
| FreeBSD | amd64 | System V | v0.5.0 | Cross-compile verified |

---

## Roadmap

| Version | Status | Highlights |
|---------|--------|------------|
| v0.2.0 | Released | Callback API, 2000-entry trampoline table |
| v0.3.x | Released | ARM64 (AAPCS64), HFA, Apple Silicon |
| v0.4.0 | Released | crosscall2 for C-thread callbacks |
| v0.4.1 | Released | ABI compliance audit — 10/11 gaps fixed |
| v0.4.2 | Released | purego compatibility (`-tags nofakecgo`) |
| **v0.5.0** | **Next** | Windows ARM64, FreeBSD, variadic functions, builder API |
| v1.0.0 | Planned | API stability (SemVer 2.0), security audit |

See [CHANGELOG.md](CHANGELOG.md) for version history and [ROADMAP.md](ROADMAP.md) for the full plan.

---

## Testing

```bash
go test ./...                          # all tests
go test -cover ./...                   # with coverage (89%)
go test -bench=. -benchmem ./ffi       # benchmarks
go test -v ./ffi                       # verbose, auto-detects platform
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Technical architecture: assembly, ABIs, callbacks |
| [docs/PERFORMANCE.md](docs/PERFORMANCE.md) | Benchmarks, optimization strategies, Go 1.26 |
| [CHANGELOG.md](CHANGELOG.md) | Version history, migration guides |
| [ROADMAP.md](ROADMAP.md) | Development roadmap to v1.0 |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contribution guidelines |
| [SECURITY.md](SECURITY.md) | Security policy |
| [examples/](examples/) | Working code examples |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

1. Fork → feature branch → tests (80%+ coverage) → lint → PR
2. Conventional commits: `feat:`, `fix:`, `docs:`, `test:`

---

## Acknowledgments

- **[purego](https://github.com/ebitengine/purego)** — proved that pure Go FFI is possible. The `crosscall2` callback mechanism, `fakecgo` approach, and assembly trampoline patterns were pioneered by purego. goffi exists because purego cleared the path.
- **[libffi](https://sourceware.org/libffi/)** — reference for FFI architecture patterns and CIF design.
- **Go runtime** — `runtime.cgocall` for GC-safe stack switching, `crosscall2` for C→Go transitions.

---

## Ecosystem

goffi powers an ecosystem of pure Go GPU libraries:

| Project | Description |
|---------|-------------|
| [go-webgpu/webgpu](https://github.com/go-webgpu/webgpu) | Zero-CGO WebGPU bindings (wgpu-native) |
| [born-ml/born](https://github.com/born-ml/born) | ML framework for Go, GPU-accelerated |
| [gogpu](https://github.com/gogpu) | GPU computing platform — dual Rust + Pure Go backends |
| [wgpu-native](https://github.com/gfx-rs/wgpu-native) | Native WebGPU implementation (upstream) |

---

## License

MIT — see [LICENSE](LICENSE).

---

*goffi v0.4.1 | [GitHub](https://github.com/go-webgpu/goffi) | [pkg.go.dev](https://pkg.go.dev/github.com/go-webgpu/goffi) | [Dev.to](https://dev.to/kolkov/goffi-zero-cgo-foreign-function-interface-for-go-how-we-call-c-libraries-without-a-c-compiler-ca5)*
