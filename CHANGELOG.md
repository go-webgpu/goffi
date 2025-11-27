# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- See [ROADMAP.md](ROADMAP.md) for upcoming features
- v0.3.0: Builder pattern API, platform-specific struct handling
- v0.5.0: ARM64 support, variadic functions
- v1.0.0: LTS release with API stability guarantee

## [0.2.0] - 2025-11-27

### Added
- **Callback support** for C-to-Go function calls (`NewCallback` API)
  - `NewCallback(fn any) uintptr` - Register Go function as C callback
  - Pre-compiled trampoline table with 2000 entries
  - Thread-safe callback registry with mutex protection
  - Reflection-based argument and return value marshaling
  - System V AMD64 ABI compatibility (Linux, macOS)
  - Win64 ABI compatibility (Windows)
  - **Files**:
    - `ffi/callback.go` - Core callback implementation
    - `ffi/callback_amd64.s` - Assembly trampolines (2000 entries)
    - `ffi/callback_test.go` - Comprehensive test suite
  - **Supported argument types**:
    - Integers: int, int8, int16, int32, int64
    - Unsigned: uint, uint8, uint16, uint32, uint64, uintptr
    - Floats: float32, float64
    - Pointers: *T, unsafe.Pointer
    - Boolean: bool
  - **Return types**: All above types + void (no return)
  - **Tests**: 20 comprehensive tests covering all scenarios

### Changed
- **Roadmap updated**: Callbacks moved from v0.5.0 to v0.2.0
- Builder pattern API moved to v0.3.0

### Use Case: WebGPU Async Operations
```go
// Create callback for wgpuInstanceRequestAdapter
cb := ffi.NewCallback(func(status int, adapter uintptr, msg uintptr, ud uintptr) {
    result := (*adapterResult)(unsafe.Pointer(ud))
    result.status = status
    result.adapter = adapter
    close(result.done)
})

// Pass to C function
ffi.CallFunction(&cif, wgpuRequestAdapter, nil,
    []unsafe.Pointer{&instance, &opts, &cb, &userdata})
```

### Known Limitations

**Callback-specific:**
- Maximum 2000 callbacks per process (memory never released)
- Complex types (string, slice, map, chan, interface) not supported as arguments
- Callbacks must have at most one return value

**Windows SEH (Go runtime limitation):**
- C++ exceptions crash the program on Windows ([Go #12516](https://github.com/golang/go/issues/12516))
- Affects Rust libraries with `panic=unwind` (default), including wgpu-native
- **This is NOT goffi-specific** - CGO has the same issue
- Workaround: Build native libraries with `panic=abort` or use Linux/macOS
- Fix planned for **Go 1.26** ([#58542](https://github.com/golang/go/issues/58542))

## [0.1.1] - 2025-11-18

### Added
- **macOS AMD64 support** via System V ABI shared implementation
  - Shared `call_unix.s` assembly for Linux and macOS (identical System V ABI)
  - Platform-specific dynamic library constants (RTLD_GLOBAL: 0x8 on macOS vs 0x100 on Linux)
  - Complete `dl_darwin.go` implementation with LoadLibrary/GetSymbol/FreeLibrary
  - `internal/dl/` Unix implementation shared between platforms
  - `fakecgo` support extended to macOS
- **Thread safety documentation** in `ffi/ffi.go`
  - Documented concurrent access patterns
  - Clarified race detector limitation for zero-CGO libraries

### Changed
- **CI/CD improvements** for cross-platform testing
  - Added `macos-13` runner (Intel AMD64) to test matrix
  - Fixed coverage calculation (test specific packages instead of all files)
  - Explicit `CGO_ENABLED=0` environment variable in all jobs
  - Coverage restored from 28-56% (diluted) to **87.1%** (accurate)
- **Architecture refactoring** for better code organization
  - Renamed `call_linux.s` → `call_unix.s` with `(linux || darwin)` build tags
  - Renamed `syscall_linux_amd64.*` → `syscall_unix_amd64.*` for shared Unix code
  - Split platform-specific constants into `dl_linux.go` and `dl_darwin.go`
  - Shared implementation in `dl_unix.go` for both Unix platforms

### Fixed
- Linter exclusions for assembly-called functions and FFI unsafe.Pointer usage
- Build constraints compatibility with fakecgo `!cgo` tag across all platforms
- CI coverage calculation methodology (test only main packages: `./ffi ./types`)

### Platform Support
- ✅ Linux AMD64 (System V ABI) - **FULLY SUPPORTED**
- ✅ Windows AMD64 (Win64 ABI) - **FULLY SUPPORTED**
- ✅ macOS AMD64 (System V ABI) - **NEWLY ADDED**
- ⏳ ARM64 (AAPCS64 ABI) - Planned for v0.5.0

### Infrastructure
- All 3 platforms tested in CI/CD (ubuntu-latest, windows-latest, macos-13)
- Quality gate: 70% minimum coverage threshold (current: 87.1%)
- Benchmark validation: FFI overhead < 200ns threshold

## [0.1.0] - 2025-11-17

### Added
- **Professional typed error system** following Go 2025 best practices
  - `InvalidCallInterfaceError`: Detailed CIF preparation errors with field, reason, and index
  - `LibraryError`: Dynamic library operation errors with operation type and underlying cause
  - `CallingConventionError`: Unsupported calling convention errors with platform info
  - `TypeValidationError`: Type descriptor validation errors with kind and index
  - `UnsupportedPlatformError`: Platform not supported errors with OS/Arch details
- **Context support** for cancellation and timeouts (`CallFunctionContext`)
- `DefaultConvention()` helper function for automatic platform detection
- `types.DefaultCall` constant for simplified cross-platform code
- `FreeLibrary()` implementation on all platforms (Windows + Linux)
- Comprehensive error handling with `errors.As()` and `errors.Is()` support
- **Comprehensive benchmarks** (CRITICAL milestone!)
  - `BenchmarkGoffiOverhead`: 88.09 ns/op (empty function)
  - `BenchmarkGoffiIntArgs`: 113.9 ns/op (integer arguments)
  - `BenchmarkGoffiStringOutput`: 97.81 ns/op (string processing)
  - `BenchmarkDirectGo`: 0.21 ns/op (baseline)
  - See [docs/PERFORMANCE.md](docs/PERFORMANCE.md) for complete analysis

### Changed
- **BREAKING**: Removed redundant `argCount` parameter from `PrepareCallInterface`
  - Old: `PrepareCallInterface(cif, convention, argCount, returnType, argTypes)`
  - New: `PrepareCallInterface(cif, convention, returnType, argTypes)`
  - Migration: Simply remove the `argCount` parameter - it's now calculated automatically
- Improved error messages with specific context (field names, indices, reasons)
- Enhanced documentation with godoc examples for all public APIs

### Fixed
- Resource leaks: Added `FreeLibrary()` to properly clean up loaded libraries
- Linter warnings: Added `//nolint` annotations for assembly-called functions

### Documentation
- Added comprehensive package documentation with usage examples
- Documented all exported functions with parameters, returns, and examples
- Added safety guidelines for `unsafe.Pointer` usage
- Created API audit documentation in `docs/dev/`
- **NEW**: [docs/PERFORMANCE.md](docs/PERFORMANCE.md) - 650+ lines comprehensive performance guide
- **NEW**: [ROADMAP.md](ROADMAP.md) - Development roadmap to v1.0.0
- **NEW**: [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- **NEW**: [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) - Community standards
- **NEW**: [SECURITY.md](SECURITY.md) - Security policy and best practices

### Testing
- Achieved **89.1% test coverage** (exceeding 80% target)
- Added 23 comprehensive test functions with 50+ test scenarios
- Tested all error types with `errors.As()` and `errors.Is()`
- Platform-specific tests for Windows and Linux
- Context cancellation and timeout tests
- **NEW**: Benchmark suite with 8 benchmarks (overhead, types, performance)

### Infrastructure
- **NEW**: `.github/CODEOWNERS` - Code ownership (@kolkov)
- **NEW**: `.github/workflows/ci.yml` - Comprehensive CI/CD (lint, format, test, benchmarks, coverage)
- **NEW**: `.codecov.yml` - Codecov configuration (70% target)
- **NEW**: `scripts/pre-release-check.sh` - Pre-release validation script

### Performance
- **FFI Overhead**: 88-114 ns/op (BETTER than estimated 230ns!)
- **Verdict**: ✅ **Excellent for WebGPU** (< 5% overhead for GPU operations)
- **Comparison**: Competitive with CGO (~200-250ns) and purego (~150-200ns)

---

## Migration Guide: v0.1.0 → v0.1.1

### No Breaking Changes

Version 0.1.1 is fully backward compatible with 0.1.0. All existing code will continue to work without modifications.

### What's New

**macOS AMD64 Support** - If you were previously targeting only Linux/Windows, you can now add macOS to your build targets:

```bash
# Build for macOS
GOOS=darwin GOARCH=amd64 go build ./...

# Your existing code works unchanged
handle, _ := ffi.LoadLibrary("libc.dylib")  # macOS system library
```

**Thread Safety Documentation** - Review new concurrency guidelines in package documentation.

---

## Migration Guide: Earlier Versions → v0.1.0

### Breaking Changes (from pre-v0.1.0)

#### 1. PrepareCallInterface Signature Change

**Old code:**
```go
err := ffi.PrepareCallInterface(
    &cif,
    types.WindowsCallingConvention,
    2,  // ❌ argCount parameter removed
    types.SInt32TypeDescriptor,
    []*types.TypeDescriptor{
        types.PointerTypeDescriptor,
        types.SInt32TypeDescriptor,
    },
)
```

**New code:**
```go
err := ffi.PrepareCallInterface(
    &cif,
    types.DefaultCall,  // ✅ Use DefaultCall for cross-platform
    types.SInt32TypeDescriptor,
    []*types.TypeDescriptor{
        types.PointerTypeDescriptor,
        types.SInt32TypeDescriptor,
    },
)
```

### Recommended Updates (Non-Breaking)

#### 1. Use DefaultCall for Cross-Platform Code

**Old:**
```go
var convention types.CallingConvention
if runtime.GOOS == "windows" {
    convention = types.WindowsCallingConvention
} else {
    convention = types.UnixCallingConvention
}
```

**New:**
```go
convention := types.DefaultCall  // Automatically resolves to platform convention
```

#### 2. Add Resource Cleanup with FreeLibrary

**Old:**
```go
handle, err := ffi.LoadLibrary("mylib.dll")
if err != nil {
    return err
}
// ❌ Library never freed - resource leak!
```

**New:**
```go
handle, err := ffi.LoadLibrary("mylib.dll")
if err != nil {
    return err
}
defer ffi.FreeLibrary(handle)  // ✅ Proper cleanup
```

#### 3. Use Context for Cancellation

**Old:**
```go
err := ffi.CallFunction(&cif, funcPtr, &result, args)
```

**New (with timeout):**
```go
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()

err := ffi.CallFunctionContext(ctx, &cif, funcPtr, &result, args)
if err == context.DeadlineExceeded {
    // Handle timeout
}
```

#### 4. Use Typed Errors for Better Error Handling

**Old:**
```go
if err != nil {
    log.Fatal(err)  // Generic error handling
}
```

**New:**
```go
var libErr *ffi.LibraryError
if errors.As(err, &libErr) {
    fmt.Printf("Failed to %s %q: %v\n", libErr.Operation, libErr.Name, libErr.Err)
}

var icErr *ffi.InvalidCallInterfaceError
if errors.As(err, &icErr) {
    if icErr.Index >= 0 {
        fmt.Printf("Argument %d failed: %s\n", icErr.Index, icErr.Reason)
    }
}
```

---

## Release Notes: v0.1.0

### What's New

**Zero-Dependency FFI** - Pure Go implementation without CGO, enabling:
- WebGPU access through wgpu-native
- Cross-platform graphics programming
- High-performance GPU operations

**Professional Error Handling** - Go 2025 best practices:
- Structured errors with context (field, index, reason)
- Type-safe error checking with `errors.As()`
- Better debugging with detailed error messages

**Simplified API** - Reduced boilerplate:
- Automatic argument counting
- Platform auto-detection with `DefaultCall`
- Resource cleanup with `FreeLibrary()`
- Context support for timeouts/cancellation

**High Quality** - Production-ready:
- 89.1% test coverage
- 0 linter issues
- Comprehensive documentation
- Professional error messages

### Performance

- ~50-60ns overhead per call
- 0 allocations for most operations
- Hand-optimized assembly for each platform
- Suitable for real-time graphics (60 FPS+)

### Known Limitations

**Critical** (affects functionality):
- **Variadic functions NOT supported** (`printf`, `sprintf`, etc.)
  - Win64 requires float→GP register duplication
  - System V requires `AL` register = SSE count
  - Workaround: Use non-variadic wrappers
  - Planned: v0.5.0

**Important** (platform-specific):
- **Struct packing** follows System V ABI only
  - Windows `#pragma pack` directives NOT honored
  - MSVC alignment may differ from GCC/Clang
  - Workaround: Manually specify `Size`/`Alignment`
  - Planned: v0.2.0 (platform-specific rules)

**Architectural**:
- **Composite types** (structs) require manual initialization via `PrepareCallInterface`
- **Cannot interrupt** C functions mid-execution (use `CallFunctionContext` for timeouts)
- **Limited to amd64** architecture (ARM64 planned for v0.5.0)
- **No bitfields** in structs (manual bit manipulation required)
- **Race detector not supported** - Race detection requires CGO_ENABLED=1, which conflicts with our fakecgo (!cgo build tag). This is a fundamental limitation of zero-CGO libraries. Manual testing possible with real C runtime.

**Performance** (BENCHMARKED in v0.1.0):
- **Measured 88-114 ns/op** FFI overhead (better than estimated 230ns!)
- **< 5% overhead** for WebGPU operations (GPU calls are 1-100µs)
- Acceptable for: WebGPU, system calls, I/O operations, GPU computing
- NOT recommended for: Tight loops (>100K calls/sec), hot-path math libraries
- See [docs/PERFORMANCE.md](docs/PERFORMANCE.md) for complete analysis

### Roadmap

See [API_TODO.md](docs/dev/API_TODO.md) for detailed roadmap to v1.0.

**v0.2.0** (Usability) - Q2 2025
- **CRITICAL**: Comprehensive benchmarks (vs CGO/purego)
- Builder pattern for CallInterface
  ```go
  lib.Call("func").Arg(...).ReturnInt()
  ```
- Platform-specific struct alignment (Windows `#pragma pack`)
- Type-safe argument helpers (`ffi.Int32()`, `ffi.String()`)
- More examples (15+ real-world use cases)

**v0.5.0** (Platform Expansion) - Q3 2025
- ARM64 support (Linux + macOS AAPCS64 ABI)
- macOS AMD64 testing/validation
- **Variadic function support** (printf, sprintf, etc.)
  - AL register handling (System V)
  - Float→GP duplication (Win64)
- Callback support (C→Go calls)
- Windows ARM64 (experimental)

**v0.8.0** (Advanced Features) - Q4 2025
- Codegen tool (`goffi-gen`) - JSON intermediate format
  ```bash
  goffi-gen --input=api.json --output=bindings.go
  ```
- Struct builder API
- Performance optimizations (JIT stub generation?)
- Thread-local storage (TLS) handling

**v1.0.0** (Stable Release) - Q1 2026
- API stability guarantee (SemVer 2.0)
- Security audit
- Reference implementations (WebGPU, Vulkan, SQLite)
- Comprehensive documentation (book-style guide)
- Performance benchmarks published
- Support policy (LTS for v1.x)

---

[Unreleased]: https://github.com/go-webgpu/goffi/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/go-webgpu/goffi/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/go-webgpu/goffi/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/go-webgpu/goffi/releases/tag/v0.1.0
