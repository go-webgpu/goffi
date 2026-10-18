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
_, _ = ffi.CallFunction(cif, sym, unsafe.Pointer(&result), args)
```

---

## Features

| | Feature | Details |
|---|---------|---------|
| **Zero CGO** | Pure Go | No C compiler needed with `CGO_ENABLED=0`. `go get` and build. |
| **Fast** | 88–114 ns/op | Pre-computed CIF; the call path allocates nothing (reuse the `avalue` slice, see [Performance](#performance)) |
| **Cross-platform** | 8 desktop targets + Android preview | Windows, Linux, macOS, FreeBSD × AMD64 + ARM64; Android arm64/API 29+ candidate pending physical-device startup proof |
| **Callbacks** | C→Go safe where validated | `crosscall2` on Linux, macOS and FreeBSD; `syscall.NewCallback` on Windows; Android callbacks fail explicitly until a physical-thread proof exists |
| **Type-safe** | Runtime validation | 5 typed error types with `errors.As()` support |
| **Struct pass/return** | Full ABI | System V: INTEGER/SSE classification, ≤8B (RAX/XMM0), 9–16B (RAX/XMM × RAX/XMM), >16B (sret). Win64: 1/2/4/8B in RAX, other sizes via hidden pointer. arm64: HFA in S/D registers, ≤16B in X0–X1, >16B via X8 |
| **Variadic** | `printf`/`sprintf` | `PrepareVariadicCallInterface` — Apple ARM64 stack-force included |
| **errno** | Always captured | Thread-safe assembly-level capture — first pure-Go FFI on Linux |
| **Context** | Timeouts | `CallFunctionContext(ctx, ...)` cancellation |
| **Race detector** | `-race` compatible | `CGO_ENABLED=1 go test -race` runs in CI on every runner with a race detector |
| **Tested** | ~90% coverage on amd64 | CI on Linux, Windows, macOS × amd64/arm64 (CGO=0 and CGO=1) and in FreeBSD 14/15 VMs |

---

## Quick Start

Android arm64/API 29+ is a preview candidate in both CGO modes. Cross-build,
ABI, and ELF probes pass, but physical-device startup proof is still pending;
see [docs/ANDROID.md](docs/ANDROID.md) for the runtime ABI, NDK probe, and the
intentional callback limitation.

### Installation

```bash
go get github.com/go-webgpu/goffi
```

### Requirements

goffi builds under **both** `CGO_ENABLED=0` and `CGO_ENABLED=1`. `CGO_ENABLED=0` needs no C compiler and is the recommended mode for the gogpu ecosystem and most users. The Go toolchain enables cgo by default when it finds a C compiler (check `go env CGO_ENABLED`), so set `CGO_ENABLED=0` explicitly if you want that mode.

```bash
# CGO_ENABLED=0 (recommended, no C compiler needed)
CGO_ENABLED=0 go build ./...

# CGO_ENABLED=1 (needs a C compiler; e.g. for go test -race, which requires cgo on Linux)
CGO_ENABLED=1 go build ./...

# FreeBSD with CGO_ENABLED=0: fakecgo must export environ/__progname for libc
CGO_ENABLED=0 go build -gcflags=github.com/go-webgpu/goffi/internal/fakecgo=-std ./...
```

> **Known issue — Linux and FreeBSD, `CGO_ENABLED=1`:** a binary that contains goffi **and other cgo code** (any package with `import "C"`, including purego), or that is linked with `-linkmode=external`, `-buildmode=c-shared` or `c-archive`, currently fails to link with `unhandled relocation … (SDYNIMPORT …)` ([#72](https://github.com/go-webgpu/goffi/issues/72); the fix is discussed in [#104](https://github.com/go-webgpu/goffi/issues/104)). macOS and Windows are not affected. Workarounds are listed under [Known Limitations](#known-limitations).

> **How?** goffi uses Go's `cgo_import_dynamic` for dynamic library loading. Under `CGO_ENABLED=0` the cgo runtime is supplied by `internal/fakecgo`; under `CGO_ENABLED=1` the standard `runtime/cgo` is linked in. Both modes share the same FFI fast path and ABIs.

### Linking modes (Linux)

`CGO_ENABLED=0` does **not** imply a fully static ELF when goffi is imported. `//go:cgo_import_dynamic` for `dlopen` / libc still records `PT_INTERP` and `DT_NEEDED` (`libdl.so.2`, `libc.so.6`, `libpthread.so.0`). This matches purego and is required for host `dlopen` — the kernel only maps `ld.so` when `PT_INTERP` is present. See [goffi#74](https://github.com/go-webgpu/goffi/issues/74) and [gogpu#474](https://github.com/gogpu/gogpu/issues/474).

| Mode | How | ELF shape | `LoadLibrary` | Typical use |
|------|-----|-----------|---------------|-------------|
| **Dynamic FFI** (default) | `CGO_ENABLED=0 go build` | dynamic + `libdl`/`libc` | yes | desktop GPU/GUI |
| **Musl dynamic** | build on Alpine / `CC=musl-gcc` | dynamic vs musl | yes | Alpine containers with GPU/GUI |
| **Static no-FFI** | `CGO_ENABLED=0 go build -tags goffi_static` | fully static (no `PT_INTERP`, no `NEEDED`) | no (`errors.Is(err, ffi.ErrStaticBuild)`) | `FROM scratch`, air-gapped CLI |

```bash
# Fully static Linux amd64/arm64 binary (FFI unavailable)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags goffi_static -o app .
file app   # statically linked
# Verify: no INTERP / NEEDED
scripts/check-elf-linking.sh --static ./app
```

Under `-tags goffi_static`, errno capture is unavailable (always returns 0): `ErrnoFnAddr()` is a no-op so the assembly trampoline skips `__errno_location` / `__error`, which need dynamic libc.

`FROM scratch` + Vulkan/Wayland/libX11 via host `dlopen` is not possible without either `ld.so` or a userspace ELF loader (see [docs/ADR-001-userspace-elf-loader.md](docs/ADR-001-userspace-elf-loader.md)). Windows is unaffected (`LoadLibraryW` via ntdll).

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

	// Call strlen — avalue elements are pointers TO argument values.
	// Keep pointer arguments in unsafe.Pointer variables, never uintptr.
	testStr := "Hello, goffi!\x00"
	strPtr := unsafe.Pointer(unsafe.StringData(testStr))
	var length uint64

	_, err = ffi.CallFunction(cif, strlen, unsafe.Pointer(&length), []unsafe.Pointer{unsafe.Pointer(&strPtr)})
	if err != nil {
		panic(err)
	}

	fmt.Printf("strlen(%q) = %d\n", testStr[:len(testStr)-1], length)
	// Output: strlen("Hello, goffi!") = 13
}
```

### Example: Calling a Variadic C Function

Use `PrepareVariadicCallInterface` instead of `PrepareCallInterface` for C variadic functions
(`printf`, `sprintf`, custom variadic APIs). Specify `nfixedargs` — the count of fixed parameters
before `...` in the C prototype. goffi automatically applies Apple's ARM64 stack-force rule on
`darwin/arm64` so the same Go code works correctly on all platforms.

```go
// C prototype: int64_t sum_variadic(int64_t count, ...)
// Call as: sum_variadic(3, 10, 20, 30) → 60

var cif types.CallInterface
err := ffi.PrepareVariadicCallInterface(
    &cif,
    types.DefaultCall,
    1, // nfixedargs: only 'count' is fixed; 10/20/30 are variadic
    types.SInt64TypeDescriptor,
    []*types.TypeDescriptor{
        types.SInt64TypeDescriptor, // count (fixed)
        types.SInt64TypeDescriptor, // arg1 (variadic)
        types.SInt64TypeDescriptor, // arg2 (variadic)
        types.SInt64TypeDescriptor, // arg3 (variadic)
    },
)

count := int64(3)
a1, a2, a3 := int64(10), int64(20), int64(30)
var result int64
_, _ = ffi.CallFunction(&cif, sym, unsafe.Pointer(&result), []unsafe.Pointer{
    unsafe.Pointer(&count),
    unsafe.Pointer(&a1),
    unsafe.Pointer(&a2),
    unsafe.Pointer(&a3),
})
// result == 60
```

A new CIF must be prepared for each unique combination of variadic argument types. The fixed-arg
portion of the CIF can be reused by re-calling `PrepareVariadicCallInterface` with different
variadic arg type slices.

### Example: Passing and Returning Structs

goffi handles C struct pass-by-value across all ABI size classes. Describe the struct layout
with a `TypeDescriptor`, then pass struct values directly via `unsafe.Pointer(&s)`.

```go
// C struct: typedef struct { int64_t x; int64_t y; } Point;
pointType := &types.TypeDescriptor{
    Kind:      types.StructType,
    Size:      16,          // must match C sizeof(Point)
    Alignment: 8,           // must match C alignof(Point)
    Members: []*types.TypeDescriptor{
        types.SInt64TypeDescriptor, // x
        types.SInt64TypeDescriptor, // y
    },
}

var cif types.CallInterface
ffi.PrepareCallInterface(&cif, types.DefaultCall,
    pointType,                                            // return: Point
    []*types.TypeDescriptor{types.SInt64TypeDescriptor, types.SInt64TypeDescriptor},
)

x, y := int64(3), int64(4)
var result Point
_, _ = ffi.CallFunction(&cif, makePointFn,
    unsafe.Pointer(&result),                              // buffer for struct return value
    []unsafe.Pointer{unsafe.Pointer(&x), unsafe.Pointer(&y)},
)
// result.X == 3, result.Y == 4

// Pass struct as argument:
var distCif types.CallInterface
ffi.PrepareCallInterface(&distCif, types.DefaultCall,
    types.SInt64TypeDescriptor,
    []*types.TypeDescriptor{pointType, pointType},        // two Point args
)

a := Point{X: 0, Y: 0}
b := Point{X: 3, Y: 4}
var dist int64
_, _ = ffi.CallFunction(&distCif, distFn,
    unsafe.Pointer(&dist),
    []unsafe.Pointer{unsafe.Pointer(&a), unsafe.Pointer(&b)}, // pointer to struct data
)
// dist == 25
```

Structs >16 bytes (on Windows AMD64: every struct whose size is not 1, 2, 4 or 8 bytes) are returned via a hidden pointer (sret) — goffi handles this transparently. Windows AMD64 passes struct arguments of other sizes as a pointer to a copy, so the callee never modifies your value.

See [`examples/struct/`](examples/struct/) for a complete working example with compile-and-run.

### avalue Convention: Pointer-to-Value

Every `avalue[i]` must be a **pointer TO the argument value**. GoFFI dereferences it to read
the actual value passed to C. This follows the [libffi convention](https://sourceware.org/libffi/).

For most cases this is straightforward — `unsafe.Pointer(&x)` for scalars, `unsafe.Pointer(&buf)` for
pointers. The non-obvious case is **out-pointer parameters** (where C writes a result through a pointer):

```go
// C: int get_device(uint32_t index, void **device)
// The function writes *device — an out-pointer parameter.

// WRONG — C receives NULL (GoFFI reads the VALUE of device, which is nil):
var device unsafe.Pointer
_, _ = ffi.CallFunction(&cif, fn, unsafe.Pointer(&ret),
    []unsafe.Pointer{unsafe.Pointer(&index), unsafe.Pointer(&device)})

// CORRECT — C receives &device (GoFFI reads the VALUE of devicePtr, which is &device):
var device unsafe.Pointer
devicePtr := unsafe.Pointer(&device)
_, _ = ffi.CallFunction(&cif, fn, unsafe.Pointer(&ret),
    []unsafe.Pointer{unsafe.Pointer(&index), unsafe.Pointer(&devicePtr)})
// After call: device holds the handle written by C
```

**Rule of thumb:** if the C parameter is `T *out` (C writes through it), you need an intermediate
`outPtr := unsafe.Pointer(&out)` and pass `unsafe.Pointer(&outPtr)` in avalue.

---

## Performance

**FFI overhead: 88–114 ns/op** (Windows AMD64, Intel i7-1255U)

| Benchmark | Time | Allocations |
|-----------|------|-------------|
| Empty function (`getpid`) | 88 ns | 0 allocs (steady state) |
| Integer argument (`abs`) | 114 ns | 0 allocs (steady state) |
| String processing (`strlen`) | 98 ns | 0 allocs (steady state) |

`syscallArgs` is heap-allocated via `sync.Pool` for callback safety (goroutine stack may move during C→Go callbacks). Pool reuse gives 0 allocs/op in steady state.

The `avalue` slice escapes to the heap (its pointers are handed to C), so a slice literal built on every call — `[]unsafe.Pointer{unsafe.Pointer(&x)}` — costs one allocation per call (8 B per argument). Build the slice once and reuse it to keep calls at 0 allocs/op.

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

**Three ABIs** — hand-written assembly for System V and AAPCS64; on Windows amd64 the Go runtime's `syscall.SyscallN` builds the Win64 frame:

| ABI | GP Registers | FP Registers | Notes |
|-----|-------------|-------------|-------|
| System V AMD64 | RDI, RSI, RDX, RCX, R8, R9 | XMM0–XMM7 | Linux, macOS, FreeBSD |
| Win64 | RCX, RDX, R8, R9 | XMM0–XMM3 | 32-byte shadow space mandatory |
| AAPCS64 | X0–X7 | D0–D7 | HFA support for ARM64 |

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full technical deep dive.

---

## Callbacks (C → Go)

WebGPU fires async callbacks from internal Metal/Vulkan threads. These threads have no goroutine — calling Go directly would crash.

goffi uses `crosscall2` (Linux, macOS, FreeBSD) and `syscall.NewCallback` (Windows) for safe C→Go transitions from any thread:

```go
// uintptr-sized arguments and one uintptr result work on every platform;
// Windows callbacks accept only that shape.
cb := ffi.NewCallback(func(status, adapter, msg, ud uintptr) uintptr {
    // Safe even when called from a C thread
    result.handle = adapter
    close(done)
    return 0
})

_, _ = ffi.CallFunction(cif, wgpuRequestAdapter, nil, args)
<-done // Wait for GPU driver callback
```

On Unix: 2000 pre-compiled trampoline entries per process (AMD64: 5 bytes/entry, ARM64: 8 bytes/entry); float32/float64 arguments and results are supported there. Windows callbacks take `uintptr`-sized arguments only (`syscall.NewCallback` rules).

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
| Callback struct args | ≤8B, 9-16B, >16B (AMD64) | Not supported (panic) | Full |
| Callback float args/returns | XMM0 / D0 (Unix amd64, arm64; Windows callbacks are `uintptr`-only) | Args yes, returns not supported (panic) | Full |
| ARM64 HFA detection | Recursive (nested structs) | Partial (bug in nested path) | Full |
| Typed errors | 5 types + errors.As() | Generic | N/A |
| Context support | Timeouts/cancellation | No | No |
| C-thread callbacks | crosscall2 (Unix), `syscall.NewCallback` (Windows) | crosscall2 | Full |
| String/bool/slice args | Raw pointers only | Auto-marshaling | Full |
| Platform breadth | 8 desktop targets + Android preview | 8 GOARCH / 20+ OS×ARCH | All |
| AMD64 overhead | 88–114 ns | Not published | ~140 ns (Go 1.26 claims ~30% reduction) |

**Choose goffi** for GPU/real-time workloads: struct passing, zero per-call overhead, callback float returns, typed errors.

**Choose purego** for general-purpose bindings: string auto-marshaling, broad architecture support, less boilerplate.

**See also:** 
- [JupiterRider/ffi](https://github.com/JupiterRider/ffi) — pure Go binding for libffi via purego. Supports struct pass/return and variadic functions; requires libffi at runtime.
- [unxed/pureffi](https://github.com/unxed/pureffi) — A 1:1 API-compatible drop-in replacement for `purego` implemented entirely on top of `goffi`. It resolves `fakecgo` linker conflicts without build tags, correctly handles macOS ARM64 variadic stack-packing ABI requirements under the hood, and provides a zero-code-change migration path.

---

## Known Limitations

**Linux and FreeBSD: `CGO_ENABLED=1` together with other cgo code does not link** ([#72](https://github.com/go-webgpu/goffi/issues/72), fix discussed in [#104](https://github.com/go-webgpu/goffi/issues/104))
- Affects binaries that also contain another `import "C"` package (purego, database drivers, …) or are linked with `-linkmode=external`, `-buildmode=c-shared` or `c-archive`. goffi alone with `CGO_ENABLED=1` links (internal linker). macOS and Windows are not affected.
- goffi + a plain cgo package: add `-ldflags=-linkmode=internal`. This does not help with purego, c-shared or c-archive.
- goffi + purego: build with `CGO_ENABLED=0 -tags nofakecgo` (see the purego entry below).

**Linux: default builds are dynamically linked** ([#74](https://github.com/go-webgpu/goffi/issues/74))
- Importing goffi records `libdl`/`libc` via `cgo_import_dynamic` even with `CGO_ENABLED=0`. Use `-tags goffi_static` for a fully static ELF (no runtime `.so` loading), or build against musl for Alpine. See [Linking modes](#linking-modes-linux).

**Windows: C++ exceptions may crash the program** ([#12516](https://github.com/golang/go/issues/12516))
- Go runtime limitation, not goffi-specific. Go 1.22+ added partial SEH support ([#58542](https://github.com/golang/go/issues/58542)), but edge cases remain.
- Workaround: build native libraries with `panic=abort`.

**Apple ARM64: variadic args always go on stack**
- Per Apple's AAPCS64 extension, variadic arguments must be passed on the stack even when GP/FP registers are available. Use `PrepareVariadicCallInterface` (not `PrepareCallInterface`) for variadic C functions on all platforms — goffi handles the Darwin-specific register flush automatically.

**Struct packing follows System V ABI only**
- Windows `#pragma pack` not honored. Manually specify `Size`/`Alignment` in `TypeDescriptor`.

**No bitfields** in struct types.

**Unix: duplicate symbol conflict with purego** ([#22](https://github.com/go-webgpu/goffi/issues/22))
- When using goffi and purego in the same binary with `CGO_ENABLED=0`, the linker reports `duplicated definition of symbol _cgo_init`. Both libraries include `internal/fakecgo` which defines identical runtime symbols.
- **Workaround A:** Build with `-tags nofakecgo` to disable goffi's fakecgo, relying on purego's copy:
  ```bash
  CGO_ENABLED=0 go build -tags nofakecgo ./...
  # FreeBSD: also -gcflags=github.com/ebitengine/purego/internal/fakecgo=-std
  ```
- **Workaround B (macOS only for now):** Build with `CGO_ENABLED=1`. In cgo mode the real `runtime/cgo` supplies the symbols and both libraries' `fakecgo` packages are gated out by `//go:build !cgo`. On Linux and FreeBSD this currently fails to link ([#72](https://github.com/go-webgpu/goffi/issues/72), see above).
- **Workaround C (Recommended for complete integration):** Use [pureffi](https://github.com/unxed/pureffi).
  `pureffi` is a 1:1, drop-in API-compatible wrapper over `goffi` that acts as a transparent replacement for `purego` via Go's `replace` directive. It completely eliminates the need for compilation tags, resolves the duplicate symbol conflict, and natively fixes Apple Silicon variadic ABI stack-packing.

---

## Platform Support

| Platform | Arch | ABI | Verification |
|----------|------|-----|--------------|
| Windows | amd64 | Win64 | CI: tests on a native runner |
| Windows | arm64 | AAPCS64 | CI: tests on a native runner |
| Linux | amd64 | System V | CI: tests on a native runner |
| Linux | arm64 | AAPCS64 | CI: tests on a native runner |
| macOS | amd64 | System V | CI: tests on a native runner |
| macOS | arm64 | AAPCS64 | CI: tests on a native runner |
| FreeBSD | amd64 | System V | CI: tests in FreeBSD 14 and 15 VMs |
| FreeBSD | arm64 | AAPCS64 | CI: cross-compiled build and test build (tests not run yet) |
| Android | arm64 | AAPCS64 (Bionic) | Guarded preview (API 29+), cross-compiled in CI |

---

## Roadmap

Planned work: [ROADMAP.md](ROADMAP.md) and the [milestones](https://github.com/go-webgpu/goffi/milestones). What each release changed: [CHANGELOG.md](CHANGELOG.md).

---

## Testing

```bash
go test ./...                          # all tests (C-backed e2e tests need a C compiler, else they skip)
go test -cover ./...                   # with coverage (CI gate: 70%)
CGO_ENABLED=1 go test -race ./...      # race detector (needs cgo and a C compiler)
go test -bench=. -benchmem ./ffi       # benchmarks
go test -v ./ffi                       # verbose, auto-detects platform
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Technical architecture: assembly, ABIs, callbacks, linking |
| [docs/PERFORMANCE.md](docs/PERFORMANCE.md) | Benchmarks, optimization strategies, Go 1.26 |
| [docs/ANDROID.md](docs/ANDROID.md) | Android ARM64 preview ABI + build notes |
| [docs/ADR-001-userspace-elf-loader.md](docs/ADR-001-userspace-elf-loader.md) | Research: optional pure-Go `.so` loader |
| [CHANGELOG.md](CHANGELOG.md) | Version history, migration guides |
| [ROADMAP.md](ROADMAP.md) | Development roadmap |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contribution guidelines |
| [SECURITY.md](SECURITY.md) | Security policy |
| [examples/](examples/) | Working code examples (`simple`, `struct`) |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

1. Fork → branch from `main` → tests (CI gate: 70% coverage) → lint → PR to `main`
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



## Star History

<a href="https://starhistory.io">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.starhistory.io/png?repos=go-webgpu/goffi&style=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.starhistory.io/png?repos=go-webgpu/goffi&style=professional" />
   <img alt="Star History Chart" src="https://api.starhistory.io/png?repos=go-webgpu/goffi" width="800" />
 </picture>
</a>

## License

MIT — see [LICENSE](LICENSE).

---

*[GitHub](https://github.com/go-webgpu/goffi) | [pkg.go.dev](https://pkg.go.dev/github.com/go-webgpu/goffi) | [Dev.to](https://dev.to/kolkov/goffi-zero-cgo-foreign-function-interface-for-go-how-we-call-c-libraries-without-a-c-compiler-ca5)*
