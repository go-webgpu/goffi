# Architecture: goffi FFI Implementation

> **How goffi calls C functions from pure Go — assembly trampolines, calling conventions, and type safety**

---

## Overview

**goffi** is a zero-dependency Foreign Function Interface (FFI) for Go. It calls C library functions without CGO by using:

- **Hand-written assembly** for System V AMD64 and AAPCS64; on Windows amd64 the Go runtime's `syscall.SyscallN` builds the Win64 frame
- **`runtime.cgocall`** for GC-safe Go→C stack switching
- **`crosscall2`** for safe C→Go callback transitions from any thread (Linux, macOS, FreeBSD); `syscall.NewCallback` on Windows
- **Runtime type validation** via `TypeDescriptor` — no codegen, no reflection

---

## Four-Layer Architecture

Every goffi call traverses four layers:

```
┌──────────────────────────────────────────────┐
│  Layer 1: Go Code                            │
│  ffi.CallFunction(cif, fn, &result, args)    │
│  Type validation, CIF pre-computation        │
└──────────────────┬───────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────┐
│  Layer 2: runtime.cgocall                    │
│  Switch to system stack (g0)                 │
│  Mark goroutine as blocked, allow GC         │
└──────────────────┬───────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────┐
│  Layer 3: Assembly Wrapper                   │
│  Load registers per ABI (GP + SSE/FP)       │
│  Call target function, capture errno,        │
│  save return values                          │
└──────────────────┬───────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────┐
│  Layer 4: C Function (external library)      │
│  Executes and returns via standard ABI       │
└──────────────────────────────────────────────┘
```

### Layer 1: Call Interface (CIF) Pre-computation

Unlike reflect-based approaches, goffi classifies arguments and computes stack layout **once** at preparation time:

```go
cif := &types.CallInterface{}
ffi.PrepareCallInterface(cif, types.DefaultCall,
    types.UInt64TypeDescriptor,                            // return type
    []*types.TypeDescriptor{types.PointerTypeDescriptor},  // arg types
)

// cif now contains:
// - Argument classification (GP register / SSE register / stack)
// - Stack size and alignment
// - Flags bitmask for assembly dispatch
// Reuse cif for all subsequent calls — zero allocation per call.
```

### Layer 2: runtime.cgocall

`runtime.cgocall` is Go's internal mechanism for calling C code safely:

1. Switches to system stack (g0)
2. Marks goroutine as "in syscall" — allows GC to proceed
3. Calls our assembly wrapper
4. Restores Go stack on return

Since v0.5.6, `syscallArgs` is heap-allocated via `sync.Pool` — goroutine stacks may move during C→Go callbacks (`copystack`), and assembly on g0 holds the args pointer across the call. All ABI-boundary structs use `structs.HostLayout` (Go 1.23+) to guarantee C-compatible memory layout.

Since v0.6.0, `CallFunction` always captures C `errno` inside the assembly trampoline — the only thread-safe window (before `exitsyscall` can migrate the goroutine). Returns `(syscall.Errno, error)`. Uses `__errno_location` (Linux) / `__error` (macOS/FreeBSD) resolved via `//go:cgo_import_dynamic`.

On Windows amd64 the call goes through `syscall.SyscallN` instead (`internal/arch/amd64/call_windows.go`); its second result carries the raw XMM0 bits, which is how float and double results are returned there.

We access it via `//go:linkname`:

```go
//go:linkname runtime_cgocall runtime.cgocall
func runtime_cgocall(fn uintptr, arg unsafe.Pointer) int32
```

### Layer 3: Platform Assembly

Hand-written assembly for System V AMD64 and AAPCS64. The function receives a struct pointer containing all arguments, loads registers, calls the target, captures errno and saves return values.

**System V AMD64** (`internal/syscall/syscall_unix_amd64.s`):

```asm
TEXT syscallN(SB), NOSPLIT|NOFRAME, $0
    // R11 = args struct pointer
    // Load 6 GP registers: RDI, RSI, RDX, RCX, R8, R9
    // Load 8 SSE registers: XMM0-XMM7
    // Push stack-spill args if needed
    CALL R10                    // call target function
    // Save RAX (int return), RDX (second return), XMM0 (float return)
```

**Win64** — no goffi assembly. `internal/arch/amd64/call_windows.go` packs the arguments into `uintptr` slots (a hidden result pointer first when the result needs one) and calls `syscall.SyscallN`, which loads RCX, RDX, R8, R9 and XMM0–XMM3 from those slots and provides the 32-byte shadow space.

**AAPCS64 ARM64** (`internal/syscall/syscall_arm64.s`, Linux, macOS, Windows and FreeBSD):

```asm
// 8 GP registers: X0-X7
// 8 FP registers: D0-D7
// HFA (Homogeneous Floating-point Aggregate) support
```

---

## Calling Conventions

| Feature | System V AMD64 | Win64 | AAPCS64 |
|---------|---------------|-------|---------|
| **GP Registers** | RDI, RSI, RDX, RCX, R8, R9 | RCX, RDX, R8, R9 | X0-X7 |
| **FP Registers** | XMM0-XMM7 | XMM0-XMM3 | D0-D7 |
| **Shadow Space** | None | 32 bytes mandatory | None |
| **Stack Alignment** | 16-byte | 16-byte | 16-byte |
| **Int Return** | RAX | RAX | X0 |
| **Float Return** | XMM0 | XMM0 | D0 |
| **Struct ≤8B** | RAX (XMM0 if all members are float/double) | RAX if 1, 2, 4 or 8 bytes, else hidden sret pointer | X0 (S0/D0 for a single float/double) |
| **Struct 9-16B** | RAX + RDX (per eightbyte class) | Hidden sret pointer | X0 + X1 |
| **Struct >16B** | Hidden sret pointer | Hidden sret pointer | Hidden sret pointer |
| **HFA** | N/A | N/A | S0-S3 / D0-D3 (up to 4 float or double members) |

---

## Struct Argument Passing

ABI rules for passing structs as arguments depend on size and platform:

### System V AMD64 (Linux, macOS, FreeBSD)

Per §3.2.3, each struct is classified by its eightbytes (8-byte chunks):

- **≤ 8 bytes**: single eightbyte. If all fields are float/double → SSE (XMM register). Otherwise → INTEGER (GP register). INTEGER wins over SSE within the same eightbyte (merge rule).
- **9-16 bytes**: two eightbytes, each classified independently. First 8 bytes → GP or XMM. Remaining bytes → GP or XMM. Both classifications use the same INTEGER-wins merge rule.
- **\> 16 bytes**: MEMORY class. Caller copies struct bytes onto the stack in 8-byte chunks.

Implementation in `internal/arch/amd64/call_unix.go`, helpers in `classification.go`:
- `isStructAllFloats(t)` — returns true if all members are float/double
- `classifyEightbyte(t, startOff, endOff)` — per-eightbyte SSE classification with merge rule

### Win64 (Windows AMD64)

- **Exactly 1, 2, 4, or 8 bytes**: passed as integer by value (same register slot), whatever the members are — struct members never go to XMM
- **All other sizes**: passed by reference — goffi copies the struct and passes a pointer to the copy, so the callee may modify it without touching the Go value (libffi `src/x86/ffiw64.c` does the same)

### AAPCS64 (ARM64)

`internal/arch/arm64/layout.go` computes where every argument goes, following AAPCS64 as libffi implements it (`allocate_to_stack` in `src/aarch64/ffi.c`):

- **HFA (Homogeneous Floating-point Aggregate)**: one to four floats or doubles (possibly nested) in consecutive V registers if all of them are free; otherwise the whole HFA is copied to the stack and no more V registers are used.
- **Other composites ≤ 16 bytes**: copied as raw bytes into consecutive X registers — floating-point members included — if enough are free; otherwise copied to the stack, and no more X registers are used.
- **\> 16 bytes**: copied to memory owned by the caller; a pointer to the copy is passed, so the callee may modify its argument without touching the Go value.
- **Stack arguments**: Linux, FreeBSD and Windows align each one to at least 8 bytes and widen integers to 64 bits. Apple platforms align a stack argument only to its natural alignment and store exactly its size, so two `int32` or `float` arguments share one 8-byte word; variadic arguments take whole 8-byte slots. Callbacks read their stack arguments with the same rules.
- **Limits**: the call path copies 56 bytes of stack arguments. `PrepareCallInterface` computes the exact stack size per OS and returns `ErrTooManyArguments` when a call does not fit.

### End-to-End Testing

The ABI is verified against real C callers and callees: `TestMain` in `ffi/struct_e2e_test.go` compiles a C test library (`testdata/structtest.c`) at test time, and the `ffi/*_e2e_test.go` files call it. Covered, among others:

- struct arguments by class: `{int32, uint32}`, `{float, float}`, `{int64, int64}`, 24-byte MEMORY structs, struct + scalar, mixed int/float structs (`struct_e2e_test.go`, `struct_mixed_e2e_test.go`);
- byte structs of 1–9, 12, 15, 16, 17, 24 and 32 bytes, as argument and as result, including a nil result buffer where a hidden result pointer is needed (`struct_sizes_e2e_test.go`);
- 9–16 byte results in each register pair (`struct_return_e2e_amd64_test.go`) and struct results of a hand-built `CallInterface` (`unprepared_cif_e2e_test.go`);
- stack arguments, narrow and `int` integers, variadic calls, the one-value-per-argument rule (`stack_args_e2e_test.go`, `narrow_int_e2e_test.go`, `int_type_e2e_test.go`, `variadic_e2e_test.go`, `avalue_count_e2e_test.go`);
- callbacks called from C: float arguments and results, stack arguments, foreign threads (`callback_float_e2e_test.go`, `callback_stack_e2e_test.go`, `callback_cthread_test.go`).

The test library is built with `$CC` (default `gcc`; the FreeBSD CI job uses the base system `cc`). The tests skip only on Android (test binaries run on the device), in a `goffi_static` build, or when there is no C compiler for the test's architecture; with `GOFFI_REQUIRE_TESTLIB=1`, which CI sets wherever such a compiler exists, a missing compiler fails the run as well. A library that fails to build or load always fails the run. CI runs them on Linux, macOS and Windows (amd64 and arm64) and on FreeBSD amd64; on windows-11-arm they skip, because the runner only has an x86-64 MinGW compiler.

---

## Callback Struct Arguments (C→Go)

When C code calls a goffi callback with struct arguments, the callback dispatch (`callbackWrap` in `callback.go`) reconstructs struct values from CPU registers and stack using `reflect.Type`:

- **≤ 8 bytes**: single eightbyte in GP or XMM register. `isStructAllFloats()` determines classification (recursive, supports nested structs).
- **9-16 bytes**: two eightbytes, each classified independently via `classifyEightbyte()` using `reflect.StructField.Offset`.
- **\> 16 bytes**: MEMORY class — C caller copies bytes onto stack. Callback reads consecutive stack slots directly. No assembly changes needed.

Classification uses `reflect.Type` (not `types.TypeDescriptor`) since callback signatures are Go functions registered via `NewCallback()`.

**Limitations**: callback struct args supported on AMD64 Unix only. ARM64 and Windows callbacks do not yet support struct arguments.

---

## Struct Return Handling

ABI rules for returning structs depend on size:

- **≤ 8 bytes**: returned in RAX (INTEGER) or XMM0 (SSE, System V only) on AMD64, X0 or S0/D0 on ARM64; written to the caller's buffer with the struct's exact size. On Win64 only 1, 2, 4 and 8 bytes come back in RAX — every other size uses a hidden result pointer passed in RCX
- **9-16 bytes** (AMD64): two eightbytes, each returned in GP or XMM per classification. Four modes:

| Struct layout | Eightbyte 0 | Eightbyte 1 | Registers | Flag |
|---|---|---|---|---|
| `{int64, int64}` | INTEGER | INTEGER | RAX + RDX | `ReturnStRaxRdx` |
| `{int64, float64}` | INTEGER | SSE | RAX + XMM0 | `ReturnStRaxXmm0` |
| `{float64, int64}` | SSE | INTEGER | XMM0 + RAX | `ReturnStXmm0Rax` |
| `{float64, float64}` | SSE | SSE | XMM0 + XMM1 | `ReturnStXmm0Xmm1` |

Classification is computed at CIF-prepare time (`classifyReturnAMD64` using `classifyEightbyte`), stored in `cif.Flags`, and dispatched in `handleReturn`. This matches libffi's `UNIX64_RET_ST_*` pattern.
- **> 16 bytes**: caller passes a hidden pointer as the first argument (sret)

Implementation in `internal/arch/amd64/implementation.go`:

```go
case types.StructType:
    size := cif.ReturnType.Size
    if size <= 8 {
        // One eightbyte: XMM0 when it is SSE class, RAX otherwise.
        // rvalue is the caller's struct: write exactly its size.
        word := retVal
        if cif.Flags == types.ReturnInXMM64 {
            word = *(*uint64)(unsafe.Pointer(&fret))
        }
        copy(unsafe.Slice((*byte)(rvalue), size), (*[8]byte)(unsafe.Pointer(&word))[:size])
        break
    }
    // 9-16B: reconstruct from the register pair selected by cif.Flags
    // (ReturnStRaxRdx, ReturnStRaxXmm0, ReturnStXmm0Rax, ReturnStXmm0Xmm1).
```

A `CallInterface` filled in by hand (without `PrepareCallInterface`) has `Flags == 0`; for a struct result it is classified on each call on a copy, so the shared value is never written. Its struct `ReturnType` must have `Size` and `Alignment` set.

---

## Callback System (C → Go)

Callbacks allow C code to call back into Go — critical for async APIs like WebGPU.

### The Problem

C threads (e.g., Metal/Vulkan internal threads) have no goroutine (`G = nil`). Calling Go code directly would crash the runtime.

### Solution: crosscall2

```
C thread (wgpu-native, Metal, Vulkan)
  │  calls trampoline (1 of 2000 pre-compiled entries)
  ▼
Assembly dispatcher
  │  saves registers, loads callback index into R12 (ARM64) or stack (AMD64)
  ▼
crosscall2 → runtime.load_g → runtime.cgocallback
  │  sets up goroutine, switches to Go stack
  ▼
Go callback function (user code)
```

### Trampoline Table

2000 pre-compiled trampoline entries per process:

**AMD64** (`callback_amd64.s`) — 5 bytes per entry:

```asm
CALL ·callbackDispatcher   // 5-byte CALL, index derived from return address
```

**ARM64** (`callback_arm64.s`) — 8 bytes per entry:

```asm
MOVD $N, R12               // load callback index
B    ·callbackDispatcher    // branch (no link — preserves LR)
```

### Usage

```go
cb := ffi.NewCallback(func(status, adapter, msg, ud uintptr) uintptr {
    // Safe even when called from a C thread
    return 0
})
// Pass cb (uintptr) as a function pointer argument to C code
```

Signature rules differ per platform. On Unix, arguments may be integers of any width, `bool`, `float32`/`float64`, pointers and (amd64) structs, with zero or one result. On Windows, `syscall.NewCallback` requires `uintptr`-sized arguments and exactly one `uintptr`-sized result; floats must be passed as bits (`math.Float64bits`). On Android, `NewCallback` panics (see [ANDROID.md](ANDROID.md)).

---

## Type System

### TypeDescriptor

All types are described at runtime via `TypeDescriptor` — no reflection, no codegen:

```go
type TypeDescriptor struct {
    Size      uint16            // Size in bytes
    Alignment uint16            // Alignment requirement
    Kind      TypeKind          // VoidType, SInt32Type, DoubleType, StructType, etc.
    Members   []*TypeDescriptor // For structs (recursive)
}
```

Predefined descriptors for all C primitive types: `VoidTypeDescriptor`, `SInt8TypeDescriptor` through `UInt64TypeDescriptor`, `FloatTypeDescriptor`, `DoubleTypeDescriptor`, `PointerTypeDescriptor`.

### Struct Types

Composite types require explicit member definitions:

```go
pointType := &types.TypeDescriptor{
    Size:      16,
    Alignment: 8,
    Kind:      types.StructType,
    Members: []*types.TypeDescriptor{
        types.DoubleTypeDescriptor, // x
        types.DoubleTypeDescriptor, // y
    },
}
```

### Validation

`PrepareCallInterface` validates all types at preparation time:

- Nil checks on the CIF, the return type and every argument type
- Every type kind must be one goffi supports
- Struct members must be present and are validated recursively; a struct with `Size == 0` gets its size and alignment computed from its members
- Argument count and stack size within platform limits (`ErrTooManyArguments`)

At call time `CallFunction` additionally requires exactly one `avalue` entry per argument type, and a non-nil function pointer.

Five typed error types for precise error handling: `InvalidCallInterfaceError`, `LibraryError`, `CallingConventionError`, `TypeValidationError`, `UnsupportedPlatformError`.

Under `-tags goffi_static`, dynamic loading is unavailable: `LoadLibrary` / `GetSymbol` return errors wrapping `ErrStaticBuild` (use `errors.Is`).

### Linking modes (Linux)

`CGO_ENABLED=0` does **not** imply a fully static ELF. Default builds still record `//go:cgo_import_dynamic` for `dlopen` / libc so host `.so` loading works:

| Mode | Build | Dynamic load | Typical use |
|------|-------|--------------|-------------|
| Dynamic FFI (default) | `CGO_ENABLED=0 go build` | yes | desktop GPU/GUI |
| Static no-FFI | `CGO_ENABLED=0 go build -tags goffi_static` | no (`ErrStaticBuild`) | `FROM scratch`, air-gapped CLI |

See README [Linking modes](../README.md#linking-modes-linux) and [ADR-001](./ADR-001-userspace-elf-loader.md). Helper: `scripts/check-elf-linking.sh`.

**`CGO_ENABLED=1` on Linux and FreeBSD** ([#72](https://github.com/go-webgpu/goffi/issues/72)): goffi's `cgo_import_dynamic` stubs work only with the Go **internal** linker. `cmd/link` rejects a relocation from Go code to a dynamically imported symbol when it links externally (`cmd/link/internal/ld/data.go`; Darwin, OpenBSD, AIX and Solaris are exempt). A goffi-only `CGO_ENABLED=1` build links internally and works; any other cgo package in the binary, `-linkmode=external`, `c-shared` or `c-archive` switches to the external linker and the link fails. The fix is discussed in [#104](https://github.com/go-webgpu/goffi/issues/104).

### avalue Indirection Convention

Following the libffi convention, `avalue[i]` is a **pointer TO the argument value**. GoFFI dereferences `avalue[i]` once to read the value placed into the register or stack slot:

```
C parameter: int x         → avalue[i] = &x         → GoFFI reads *&x = x value
C parameter: char *buf     → avalue[i] = &buf        → GoFFI reads *&buf = buf address
C parameter: int *out      → avalue[i] = &outPtr     → GoFFI reads *&outPtr = &out
                              where outPtr = &out         (C receives address of out)
```

The third case (out-pointer) requires an intermediate Go variable. Without it, GoFFI would read the current value of `out` (likely zero) instead of its address, and the C function receives NULL.

---

## Variadic Function Support

### Overview

C variadic functions (`printf`, `sprintf`, `sum(count, ...)`) require a different CIF preparation path because the fixed and variadic argument regions may be handled differently by the hardware ABI.

Use `PrepareVariadicCallInterface` in place of `PrepareCallInterface`:

```go
// C prototype: int64_t sum_variadic(int64_t count, ...)
var cif types.CallInterface
err := ffi.PrepareVariadicCallInterface(
    &cif,
    types.DefaultCall,
    1, // nfixedargs: only 'count' is fixed
    types.SInt64TypeDescriptor,
    []*types.TypeDescriptor{
        types.SInt64TypeDescriptor, // count (fixed)
        types.SInt64TypeDescriptor, // variadic arg 1
        types.SInt64TypeDescriptor, // variadic arg 2
    },
)
```

A new CIF must be prepared for each unique combination of variadic argument types, matching `libffi`'s `ffi_prep_cif_var()` requirement.

### Platform Differences

**System V AMD64 (Linux, macOS Intel, FreeBSD) and AAPCS64 (Linux, FreeBSD):**  
Variadic arguments go in the same registers as fixed arguments, up to the register count limit. `PrepareVariadicCallInterface` on these platforms is functionally identical to `PrepareCallInterface` — `FixedArgCount` is stored in `CallInterface` but the argument marshalling loop is unchanged.

**Windows (AMD64/ARM64):**  
Integer variadic arguments use the same registers as fixed ones (RCX, RDX, R8, R9 on AMD64; X0–X7 on ARM64). `PrepareVariadicCallInterface` behaves identically to `PrepareCallInterface`.

The end-to-end tests (`ffi/variadic_e2e_test.go`, `cmd/variadic-test`) cover integer variadic arguments.

**Apple ARM64 (macOS/iOS, `GOOS=darwin`, `GOARCH=arm64`):**  
Apple's AAPCS64 extension mandates that **variadic arguments must be passed on the stack**, even when GP or FP registers are still available. This differs from the standard AAPCS64 used on Linux ARM64, where variadic args may be placed in X1-X7.

Implementation in `internal/arch/arm64/layout.go` (`layoutArgs`):

```go
// Apple: from the first variadic argument on, everything goes on the
// stack in 8-byte slots.
if stack.apple && cif.FixedArgCount > 0 && i == cif.FixedArgCount {
    ngrn, nsrn = nRegs, nRegs
    stack.variadic = true
}
```

This matches the behaviour of Apple's `clang` and `libffi`'s `ffi_prep_cif_var()` on Darwin ARM64.

### CallInterface.FixedArgCount

The `FixedArgCount` field in `CallInterface` stores the variadic boundary:

- `FixedArgCount == 0` — non-variadic CIF (zero value, backward compatible)
- `FixedArgCount > 0` — number of fixed parameters; args at index `FixedArgCount` and beyond are variadic

### Verification

Run `cmd/variadic-test` on Apple Silicon to confirm:

```bash
go run github.com/go-webgpu/goffi/cmd/variadic-test
# Platform: darwin/arm64
#   sum_variadic(3, 10, 20, 30) = 60 (want 60) OK
#   variadic_two_fixed(100, 200, 300) = 600 (want 600) OK
# PASS — variadic functions work on this platform
```

---

## Platform Support

| Platform | Architecture | ABI | Status |
|----------|-------------|-----|--------|
| **Linux** | AMD64 | System V | Production |
| **Windows** | AMD64 | Win64 | Production |
| **Windows** | ARM64 | AAPCS64 | Production (tested on Snapdragon X) |
| **macOS** | AMD64 | System V | Production |
| **macOS** | ARM64 | AAPCS64 | Production (tested on M3 Pro) |
| **FreeBSD** | AMD64 | System V | Production (CI: tests in FreeBSD 14 and 15 VMs) |
| **FreeBSD** | ARM64 | AAPCS64 | Cross-compiled in CI (build and test build; tests not run yet) |
| **Linux** | ARM64 | AAPCS64 | Production |
| **Android** | ARM64 | AAPCS64 (Bionic) | Guarded preview (API 29+, NDK r29) |

---

## Key Files

| File | Purpose |
|------|---------|
| `ffi/ffi.go` | Public API: `PrepareCallInterface`, `CallFunction` |
| `ffi/cif.go` | CIF preparation, type validation, stack calculation |
| `ffi/call.go` | Delegation to platform-specific implementations |
| `ffi/errors.go` | 5 typed error types |
| `ffi/callback.go` | AMD64 Unix callback trampolines (2000 entries) |
| `ffi/callback_arm64.go` | ARM64 callback trampolines (2000 entries) |
| `ffi/callback_windows.go` | Windows callbacks via `syscall.NewCallback` |
| `types/types.go` | TypeDescriptor, CallingConvention, constants |
| `internal/arch/amd64/classification.go` | Argument/return type classification |
| `internal/arch/amd64/implementation.go` | Return value handling (`handleReturn`) |
| `internal/arch/amd64/call_unix.go` | Unix AMD64 argument packing |
| `internal/arch/amd64/call_windows.go` | Win64 calls via `syscall.SyscallN` |
| `internal/arch/arm64/implementation.go` | ARM64 AAPCS64 return handling |
| `internal/arch/arm64/classification.go` | HFA detection, ARM64 classification |
| `internal/arch/arm64/layout.go` | AAPCS64 / Apple argument layout (calls and callbacks) |
| `internal/arch/arm64/call_arm64.go` | ARM64 argument packing |
| `internal/syscall/syscall_unix_amd64.s` | System V AMD64 assembly (call + errno capture) |
| `internal/syscall/syscall_arm64.s` | ARM64 assembly (Linux, macOS, Windows, FreeBSD) |
| `internal/dl/` | `dlopen` / `dlsym` / `dlerror` bindings (Unix) |
| `internal/fakecgo/` | cgo runtime for `CGO_ENABLED=0` (Unix) |
| `ffi/callback_amd64.s`, `ffi/callback_arm64.s` | Callback trampolines and dispatchers |

---

## References

1. [System V AMD64 ABI](https://gitlab.com/x86-psABIs/x86-64-ABI)
2. [Win64 Calling Convention](https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention)
3. [AAPCS64 (ARM64)](https://github.com/ARM-software/abi-aa/blob/main/aapcs64/aapcs64.rst)
4. [Go runtime: cgocall.go](https://github.com/golang/go/blob/master/src/runtime/cgocall.go)
5. [purego](https://github.com/ebitengine/purego) — inspiration for CGO-free approach
6. [libffi](https://sourceware.org/libffi/) — reference for FFI architecture patterns

---

*Last updated: 2026-10-18*
