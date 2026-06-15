# Architecture: goffi FFI Implementation

> **How goffi calls C functions from pure Go — assembly trampolines, calling conventions, and type safety**

---

## Overview

**goffi** is a zero-dependency Foreign Function Interface (FFI) for Go. It calls C library functions without CGO by using:

- **Hand-written assembly** for each platform ABI
- **`runtime.cgocall`** for GC-safe Go→C stack switching
- **`crosscall2`** for safe C→Go callback transitions (any thread)
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
│  Call target function, save return values    │
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

Since v0.5.4, the `runtime_cgocall` linkname declaration has `//go:noescape`, keeping `syscallArgs` on the goroutine stack (zero heap allocations). All ABI-boundary structs use `structs.HostLayout` (Go 1.23+) to guarantee C-compatible memory layout.
4. Restores Go stack on return

We access it via `//go:linkname`:

```go
//go:linkname runtime_cgocall runtime.cgocall
func runtime_cgocall(fn uintptr, arg unsafe.Pointer) int32
```

### Layer 3: Platform Assembly

Hand-written assembly for each ABI. The function receives a struct pointer containing all arguments, loads registers, calls the target, and saves return values.

**System V AMD64** (`syscall_unix_amd64.s`):

```asm
TEXT syscallN(SB), NOSPLIT|NOFRAME, $0
    // R11 = args struct pointer
    // Load 6 GP registers: RDI, RSI, RDX, RCX, R8, R9
    // Load 8 SSE registers: XMM0-XMM7
    // Push stack-spill args if needed
    CALL R10                    // call target function
    // Save RAX (int return), RDX (second return), XMM0 (float return)
```

**Win64** (`syscall_windows_amd64.s`):

```asm
// 4 GP registers: RCX, RDX, R8, R9
// 4 SSE registers: XMM0-XMM3
// 32-byte shadow space mandatory
```

**AAPCS64 ARM64** (`syscall_unix_arm64.s`):

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
| **Struct ≤8B** | RAX | RAX | X0 |
| **Struct 9-16B** | RAX + RDX | N/A (sret) | X0 + X1 |
| **Struct >16B** | Hidden sret pointer | Hidden sret pointer | Hidden sret pointer |
| **HFA** | N/A | N/A | D0-D3 (up to 4 floats) |

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

- **Exactly 1, 2, 4, or 8 bytes**: passed as integer by value (same register slot)
- **All other sizes**: passed by reference — caller passes a pointer

### AAPCS64 (ARM64)

- **≤ 16 bytes**: passed in GP registers (up to 2)
- **HFA (Homogeneous Floating-point Aggregate)**: up to 4 same-type floats → D0-D3
- **\> 16 bytes**: passed by reference

### End-to-End Testing

Struct argument passing is verified by `ffi/struct_e2e_test.go`, which compiles a C test library (`testdata/structtest.c`) via gcc at test time. Five scenarios are tested:

1. **≤8B integer pair** (`{int32, uint32}`) — INTEGER class, single GP register
2. **≤8B float pair** (`{float, float}`) — SSE class, single XMM register
3. **16B integer pair** (`{int64, int64}`) — two INTEGER eightbytes, two GP registers
4. **24B triple** (`{int64, int64, int64}`) — MEMORY class, copied to stack
5. **Struct + scalar** — mixed register allocation

Tests run on Linux, macOS, FreeBSD, and Windows where gcc is available; skipped gracefully otherwise.

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

- **≤ 8 bytes**: returned in RAX (INTEGER) or XMM0 (SSE) on AMD64, X0 or D0 on ARM64
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
    switch {
    case size <= 8:
        *(*uint64)(rvalue) = retVal
    case size <= 16:
        *(*uint64)(rvalue) = retVal           // RAX → bytes 0-7
        remaining := size - 8
        src := (*[8]byte)(unsafe.Pointer(&retVal2))
        dst := (*[8]byte)(unsafe.Add(rvalue, 8))
        copy(dst[:remaining], src[:remaining]) // RDX → bytes 8-15
    }
```

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
cb := ffi.NewCallback(func(status uint32, adapter uintptr, msg uintptr, ud uintptr) {
    // Safe even when called from a C thread
})
// Pass cb (uintptr) as a function pointer argument to C code
```

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

- Nil checks on all descriptors
- Size > 0 for non-void types
- Struct members recursively validated
- Alignment power-of-two check
- Argument count within platform limits

Five typed error types for precise error handling: `InvalidCallInterfaceError`, `LibraryError`, `CallingConventionError`, `TypeValidationError`, `UnsupportedPlatformError`.

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

**System V AMD64 (Linux, macOS Intel, FreeBSD):**  
Standard AAPCS64 and System V ABI pass variadic arguments in the same registers as fixed arguments, up to the register count limit. `PrepareVariadicCallInterface` on these platforms is functionally identical to `PrepareCallInterface` — `FixedArgCount` is stored in `CallInterface` but the argument marshalling loop is unchanged.

**Win64 (Windows AMD64/ARM64):**  
Same as System V for integer arguments — variadic args use the same 4 GP registers. `PrepareVariadicCallInterface` behaves identically to `PrepareCallInterface`.

**Apple ARM64 (macOS/iOS, `GOOS=darwin`, `GOARCH=arm64`):**  
Apple's AAPCS64 extension mandates that **variadic arguments must be passed on the stack**, even when GP or FP registers are still available. This differs from the standard AAPCS64 used on Linux ARM64, where variadic args may be placed in X1-X7.

Implementation in `internal/arch/arm64/call_arm64.go` (`Execute` method):

```go
// At the fixed/variadic boundary on Apple ARM64, exhaust both
// register allocators so all variadic args land on the stack.
if cif.FixedArgCount > 0 && runtime.GOOS == "darwin" && idx == cif.FixedArgCount {
    gprIdx = 8 // exhaust GP registers (X0-X7)
    fprIdx = 8 // exhaust FP registers (D0-D7)
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
| **FreeBSD** | AMD64 | System V | Cross-compile verified |
| **FreeBSD** | ARM64 | AAPCS64 | Cross-compile verified |
| **Linux** | ARM64 | AAPCS64 | Production |

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
| `internal/arch/amd64/call_unix.go` | Unix AMD64 execution |
| `internal/arch/arm64/implementation.go` | ARM64 AAPCS64 implementation |
| `internal/arch/arm64/classification.go` | HFA detection, ARM64 classification |
| `internal/syscall/syscall_unix_amd64.s` | System V AMD64 assembly |
| `internal/syscall/syscall_windows_amd64.s` | Win64 assembly |
| `internal/syscall/syscall_unix_arm64.s` | ARM64 assembly |

---

## References

1. [System V AMD64 ABI](https://gitlab.com/x86-psABIs/x86-64-ABI)
2. [Win64 Calling Convention](https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention)
3. [AAPCS64 (ARM64)](https://github.com/ARM-software/abi-aa/blob/main/aapcs64/aapcs64.rst)
4. [Go runtime: cgocall.go](https://github.com/golang/go/blob/master/src/runtime/cgocall.go)
5. [purego](https://github.com/ebitengine/purego) — inspiration for CGO-free approach
6. [libffi](https://sourceware.org/libffi/) — reference for FFI architecture patterns

---

*Current version: v0.4.1 | Last updated: 2026-03-02*
