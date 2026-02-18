# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.9] - 2026-02-18

### Fixed
- **ARM64 callback trampoline: BL overwrites LR** (Critical, [#15](https://github.com/go-webgpu/goffi/issues/15))
  - `BL` (Branch-with-Link) was destroying C caller's return address in LR register
  - Rewrote all 2000 trampoline entries to `MOVD $index, R12` + `B` (Branch without Link)
  - Matches Go runtime (`zcallback_windows_arm64.s`) and purego (`zcallback_arm64.s`) patterns
  - New dispatcher saves/restores R27 (callee-saved, used by Go assembler) and R30 (LR)
  - 176-byte stack frame, 16-byte aligned per AAPCS64
  - `entrySize` comment updated: `MOVD (4 bytes) + B (4 bytes) = 8 bytes`

- **Callback assembly symbol collision with purego** ([#15](https://github.com/go-webgpu/goffi/issues/15))
  - Global symbols `callbackasm`/`callbackasm1` conflicted with purego when linked together
  - Renamed to package-scoped middot symbols (`·callbackTrampoline`/`·callbackDispatcher`)
  - At link time these become `github.com/go-webgpu/goffi/ffi.callbackTrampoline` — no collision
  - Go variable/function names updated for consistency:
    - `callbackasmAddr` → `trampolineEntryAddr`
    - `callbackasmABI0` → `trampolineBaseAddr`

### Known Limitations
- **Callback dispatcher bypasses crosscall2** ([#16](https://github.com/go-webgpu/goffi/issues/16))
  - Callbacks work correctly on Go-managed threads (primary WebGPU use case)
  - Callbacks on C-library-created threads will crash (G = nil)
  - Planned fix: crosscall2 integration in v0.4.0

### Planned
- See [ROADMAP.md](ROADMAP.md) for upcoming features

## [0.3.8] - 2026-01-24

### Fixed
- **CGO_ENABLED=1 build error handling** ([gogpu/wgpu#43](https://github.com/gogpu/wgpu/issues/43))
  - Users on Linux/macOS with gcc/clang installed got confusing linker errors
  - Root cause: Assembly files compiled under CGO=1 but referenced undefined symbols
  - Solution: Enterprise-grade error handling with compile-time + runtime guards

### Added
- **Compile-time CGO detection** with descriptive error identifier
  - `undefined: GOFFI_REQUIRES_CGO_ENABLED_0` - immediately clear what's wrong
  - Godoc comment in `cgo_unsupported.go` with full instructions
  - Runtime panic fallback with detailed fix instructions

- **Requirements section in README.md**
  - Clear documentation that `CGO_ENABLED=0` is required
  - Three options for setting CGO_ENABLED
  - Explanation of why this is needed (cgo_import_dynamic mechanism)

### Changed
- `internal/dl/dl_stubs_unix.s` - Added `!cgo` build constraint
- `internal/dl/dl_wrappers_unix.s` - Added `!cgo` build constraint
- `internal/dl/dl_stubs_arm64.s` - Added `!cgo` build constraint
- `internal/dl/dl_wrappers_arm64.s` - Added `!cgo` build constraint
- `ffi/dl_unix.go` - Added `!cgo` build constraint
- `ffi/dl_darwin.go` - Added `!cgo` build constraint

### Technical Details
- goffi uses `cgo_import_dynamic` for dynamic library loading without CGO
- This mechanism only works when `CGO_ENABLED=0`
- On Linux/macOS with C compiler installed, Go defaults to `CGO_ENABLED=1`
- New error handling provides clear guidance instead of cryptic linker errors

## [0.3.7] - 2026-01-03

### Added
- **ARM64 Darwin comprehensive support** (PR #9 by @ppoage)
  - Tested on Apple Silicon M3 Pro (64 ns/op benchmark)
  - Nested struct handling via `placeStructRegisters()`
  - Mixed int/float struct support via `countStructRegUsage()`
  - `ensureStructLayout()` for auto-computing size/alignment
  - Assembly shim (`abi_capture_test.s`) for ABI verification
  - Comprehensive darwin ObjC tests (747 lines)
  - Struct argument tests (537 lines)

- **r2 (X1) return for 9-16 byte struct returns**
  - `Call8Float` now returns both X0 and X1
  - Fixes struct returns between 9-16 bytes on ARM64

- **uint64 bit patterns for float registers**
  - Cleaner handling of mixed float32/float64 arguments
  - `fpr [8]uint64` instead of `fpr [8]float64`

### Fixed
- **BenchmarkGoffiStringOutput segfault on darwin**
  - Pointer argument now correctly passed as `unsafe.Pointer(&strPtr)`
  - Follows documented API: args contains pointers to argument storage

### Changed
- `internal/syscall/syscall_unix_arm64.go`
  - `Call8Float` signature: `(r1, r2 uintptr, fret [4]uint64)`
  - Float registers now use raw bit patterns (uint64)

- `internal/arch/arm64/classification.go`
  - `isHomogeneousFloatAggregate` now walks nested structs
  - Returns element kind for proper HFA detection

- `internal/arch/arm64/implementation.go`
  - `handleReturn` accepts both retLo and retHi
  - `handleHFAReturn` uses bit patterns for correct float32/float64

### Contributors
- @ppoage - ARM64 Darwin fixes, ObjC tests, assembly shim

## [0.3.6] - 2025-12-29

### Fixed
- **ARM64 HFA (Homogeneous Floating-point Aggregate) returns** (Critical)
  - NSRect (4 × float64) returned zeros on Apple Silicon ([gogpu#24](https://github.com/gogpu/gogpu/issues/24))
  - Root cause: Assembly only saved D0-D1, HFA needs D0-D3
  - Solution: Save all 4 float registers (D0-D3) for HFA returns
  - Affects Objective-C runtime calls on macOS ARM64 (Apple Silicon M1/M2/M3/M4)

- **ARM64 large struct return via X8 (sret)** (Critical)
  - Non-HFA structs >16 bytes returned via implicit pointer in X8
  - Root cause: X8 register never loaded before function call
  - Solution: Load rvalue pointer into X8 for sret calls

### Added
- `ReturnHFA2`, `ReturnHFA3`, `ReturnHFA4` return flag constants
- `handleHFAReturn` function for processing HFA struct returns
- `classifyReturnARM64` now detects HFA structs (1-4 floats/doubles)
- Unit tests for ARM64 HFA classification

### Changed
- `internal/syscall/syscall_unix_arm64.go`
  - Added fr1-fr4 fields for D0-D3 float returns
  - Added r8 field for X8 sret pointer
  - Updated `Call8Float` signature: accepts r8, returns [4]float64

- `internal/syscall/syscall_unix_arm64.s`
  - Load X8 from r8 field (offset 184) before function call
  - Save D0-D3 to fr1-fr4 (offsets 152-176) after call
  - Fixed offsets: was incorrectly saving to input arg offsets

- `internal/arch/arm64/implementation.go`
  - `handleReturn` now accepts fret [4]float64 for HFA
  - Fixed sret handling: do nothing (callee writes directly via X8)

- `internal/arch/arm64/call_unix.go`
  - Pass rvalue as r8 for ReturnViaPointer calls
  - Pass full fret [4]float64 to handleReturn

### Technical Details
- AAPCS64 (ARM64 ABI): HFA structs with 1-4 same-type floats return in D0-D3
- AAPCS64: Large non-HFA structs (>16 bytes) return via hidden pointer in X8
- NSRect = CGRect = 4 × float64 = 32 bytes = HFA (returns in D0-D3)
- Fixes blank window issue on macOS ARM64 (GPU window size was 0×0)

## [0.3.5] - 2025-12-27

### Fixed
- **Windows stack arguments not implemented** (Critical)
  - Functions with >4 arguments caused `panic: stack arguments not implemented`
  - Win64 ABI: first 4 args in registers (RCX/RDX/R8/R9), args 5+ on stack
  - Solution: Use `syscall.SyscallN` with variadic args for unlimited argument support
  - Affected Vulkan functions: `vkCreateGraphicsPipelines` (6 args), `vkCmdBindVertexBuffers` (5 args), etc.
  - Reported via go-webgpu/wgpu project

### Changed
- **Simplified Windows FFI** - removed intermediate syscall wrapper
  - Removed: `internal/syscall/syscall_windows_amd64.go` (no longer needed)
  - `call_windows.go` now calls `syscall.SyscallN` directly with `args...`
  - Cleaner code, fewer indirections

### Technical Details
- `syscall.SyscallN(fn, args...)` supports up to 15+ arguments
- Handles both register (1-4) and stack (5+) arguments automatically
- Same approach used by purego for Windows FFI

## [0.3.4] - 2025-12-27

### Fixed
- **Windows stack overflow on Vulkan API calls** (Critical)
  - `callWin64` assembly used `NOSPLIT, $32` which prevented stack growth
  - When calling C functions needing significant stack (Vulkan drivers), caused `STACK_OVERFLOW` (Exception 0xc00000fd)
  - Solution: Replace direct assembly with `syscall.SyscallN` (Go runtime's asmstdcall)
  - This matches purego's proven approach for Windows FFI
  - Reported via go-webgpu/wgpu project

### Changed
- **Windows FFI architecture** - Enterprise-grade refactoring
  - Removed: `internal/arch/amd64/call_windows.s` (direct assembly)
  - Added: `internal/syscall/syscall_windows_amd64.go` (SyscallN wrapper)
  - Uses Go runtime's built-in stack management via `syscall.SyscallN`
  - Proper shadow space and stack alignment handled by Go runtime

### Technical Details
- Win64 ABI: First 4 args in RCX/RDX/R8/R9 (or XMM0-3 for floats)
- `syscall.SyscallN` internally uses `cgocall(asmstdcallAddr, ...)`
- Go runtime allocates proper stack and handles preemption/GC
- Float return values not captured (known limitation, matches purego)

## [0.3.3] - 2025-12-24

### Fixed
- **PointerType argument passing bug** ([#4](https://github.com/go-webgpu/goffi/issues/4))
  - PointerType was passing address instead of value
  - Now correctly dereferences: `*(*uintptr)(avalue[idx])` instead of `uintptr(avalue[idx])`
  - Fixed in all three Execute implementations:
    - `internal/arch/arm64/call_unix.go`
    - `internal/arch/amd64/call_unix.go`
    - `internal/arch/amd64/call_windows.go`
  - Added missing SInt8/UInt8/SInt16/UInt16 type handling in AMD64 Unix
  - Fixed float32 handling in Windows (was treating as float64)
  - Reported by go-webgpu project via GitHub Issue #4

### Added
- **Regression tests** for argument passing
  - `TestPointerArgumentPassing` - strlen-based test for PointerType (Issue #4)
  - `TestIntegerArgumentTypes` - abs-based test for integer types
  - Both tests use documented API pattern: `[]unsafe.Pointer{unsafe.Pointer(&arg)}`

### Technical Details
- API contract (ffi.go line 43): `avalue` contains pointers TO argument values
- For PointerType: pass `unsafe.Pointer(&ptr)`, not `ptr` directly
- Tests now correctly use documented pattern, preventing future regressions

## [0.3.2] - 2025-12-23

### Fixed
- **ARM64 HFA (Homogeneous Floating-point Aggregate) classification bug**
  - HFA structs (e.g., NSRect with 4 doubles) were incorrectly passed by reference
  - Now correctly passed in floating-point registers D0-D7 per AAPCS64 ABI
  - Fix: Check HFA status before struct size in `classifyArgumentARM64()`
  - Reported via go-webgpu/webgpu macOS ARM64 testing

### Technical Details
- AAPCS64 requires HFA detection before size-based classification
- Example: `NSRect` (4 × float64 = 32 bytes) is HFA → uses D0-D3, not reference
- Affects Objective-C runtime calls on Apple Silicon (M1/M2/M3/M4)

## [0.3.1] - 2025-11-28

### Fixed
- **ARM64 build constraints** for dynamic library loading functions
  - `dl_unix.go`: Added ARM64 support (`linux && (amd64 || arm64)`)
  - `dl_darwin.go`: Added ARM64 support (`darwin && (amd64 || arm64)`)
  - `stubs/caller.go`: Exclude ARM64 from stubs (`!amd64 && !arm64`)
  - Fixes `undefined: ffi.LoadLibrary` on ARM64 platforms
  - Reported by go-webgpu project

## [0.3.0] - 2025-11-28

### Added
- **ARM64 architecture support** (AAPCS64 ABI for Linux and macOS)
  - `internal/arch/arm64/` - Complete ARM64 implementation
  - `internal/syscall/syscall_unix_arm64.s` - ARM64 assembly for FFI calls
  - `ffi/callback_arm64.go` + `ffi/callback_arm64.s` - 2000-entry callback trampolines
  - X0-X7 integer registers, D0-D7 floating-point registers
  - Homogeneous Floating-point Aggregate (HFA) detection
  - Cross-compile verified: `GOOS=linux/darwin GOARCH=arm64`
- **Pre-release script improvements** for ARM64 and cross-platform builds

### Platform Support
- ✅ Linux AMD64 (System V ABI)
- ✅ Windows AMD64 (Win64 ABI)
- ✅ macOS AMD64 (System V ABI)
- 🟡 Linux ARM64 (AAPCS64 ABI) - cross-compile verified, hardware testing pending
- 🟡 macOS ARM64 (AAPCS64 ABI) - cross-compile verified, hardware testing pending

### Note
ARM64 support is feature-complete but awaiting real hardware testing. Cross-compilation
verified on all platforms. Use with caution on production ARM64 systems until v0.3.1.

## [0.2.1] - 2025-11-27

### Fixed
- **Windows callback ABI**: Use `syscall.NewCallback` for proper Win64 ABI compliance
  - Windows callbacks now use the official Go syscall mechanism
  - Resolves ABI mismatch issues with native Windows calling convention

### Documentation
- **SEH limitation documented**: C++ exceptions crash the program on Windows
  - This is a Go runtime limitation ([#12516](https://github.com/golang/go/issues/12516))
  - Affects both CGO and goffi equally
  - Fix planned for Go 1.26 ([#58542](https://github.com/golang/go/issues/58542))

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
- 🟡 ARM64 (AAPCS64 ABI) - In development for v0.3.0

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
- **Limited to amd64** architecture (ARM64 in development for v0.3.0)
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

**v0.3.0** (ARM64 Support) - Q1 2025
- ARM64 support (Linux + macOS AAPCS64 ABI)
- AAPCS64 calling convention implementation
- 2000-entry callback trampolines for ARM64
- Cross-compile verified, real hardware testing pending

**v0.5.0** (Usability + Variadic) - Q2 2025
- Builder pattern for CallInterface
  ```go
  lib.Call("func").Arg(...).ReturnInt()
  ```
- Platform-specific struct alignment (Windows `#pragma pack`)
- **Variadic function support** (printf, sprintf, etc.)
  - AL register handling (System V)
  - Float→GP duplication (Win64)
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

[Unreleased]: https://github.com/go-webgpu/goffi/compare/v0.3.8...HEAD
[0.3.9]: https://github.com/go-webgpu/goffi/compare/v0.3.8...v0.3.9
[0.3.8]: https://github.com/go-webgpu/goffi/compare/v0.3.7...v0.3.8
[0.3.7]: https://github.com/go-webgpu/goffi/compare/v0.3.6...v0.3.7
[0.3.6]: https://github.com/go-webgpu/goffi/compare/v0.3.5...v0.3.6
[0.3.5]: https://github.com/go-webgpu/goffi/compare/v0.3.4...v0.3.5
[0.3.4]: https://github.com/go-webgpu/goffi/compare/v0.3.3...v0.3.4
[0.3.3]: https://github.com/go-webgpu/goffi/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/go-webgpu/goffi/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/go-webgpu/goffi/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/go-webgpu/goffi/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/go-webgpu/goffi/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/go-webgpu/goffi/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/go-webgpu/goffi/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/go-webgpu/goffi/releases/tag/v0.1.0
