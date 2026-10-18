# Security Policy

## Supported Versions

goffi is pre-1.0. Security fixes are released as a patch of the latest minor release line; older lines are not patched.

| Version | Supported          |
| ------- | ------------------ |
| 0.6.x   | :white_check_mark: |
| < 0.6.0 | :x:                |

Stable releases (v1.0+) will follow semantic versioning with long-term support.

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in goffi, please report it responsibly.

### How to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Report it privately through a **GitHub Security Advisory**:
https://github.com/go-webgpu/goffi/security/advisories/new

### What to Include

Please include the following information in your report:

- **Description** of the vulnerability
- **Steps to reproduce** the issue (include minimal code example if possible)
- **Affected versions** (which versions are impacted)
- **Platform** (OS, architecture, `CGO_ENABLED` value, Go version)
- **Potential impact** (memory corruption, race condition, crash, etc.)
- **Suggested fix** (if you have one)

### Response Timeline

We aim to:
1. Acknowledge receipt within 72 hours
2. Provide an initial assessment within 1 week
3. Work with you on a coordinated disclosure timeline
4. Credit you in the security advisory (unless you prefer to remain anonymous)

## Security Considerations for FFI Libraries

goffi provides a Foreign Function Interface (FFI) to C libraries without CGO. Calling C means leaving Go's memory safety: the C side can read or write anything it is given, and a wrong type description is not caught by the compiler.

### 1. Memory Safety Risks

**Risk**: FFI involves passing Go pointers to C code and vice versa, which can lead to memory corruption if handled incorrectly.

**Attack Vectors**:
- Invalid pointer addresses passed to C functions
- Buffer overflows when C code writes beyond Go-allocated memory
- C code keeping a Go pointer after the call returns (the GC may free that memory)
- Dangling pointers when C library frees memory unexpectedly
- Type confusion between Go and C representations

**Mitigation in Library**:
- ✅ Type descriptor validation before calls
- ✅ Exactly one argument value required per argument type
- ✅ Calls go through `runtime.cgocall` (GC-safe stack switch)
- ✅ ABI behaviour checked by C-backed end-to-end tests on native CI runners
- ✅ CI runs the test suite with `-race`, which also enables `checkptr` pointer checks
- 🔄 Fuzzing (planned)

**User Recommendations**:
```go
// ❌ BAD - pointer hidden in a uintptr: if the goroutine stack moves,
// the uintptr keeps the old address and C reads stale memory.
buf := make([]byte, 64)
addr := uintptr(unsafe.Pointer(&buf[0]))
_, _ = ffi.CallFunction(&cif, fn, nil, []unsafe.Pointer{unsafe.Pointer(&addr)})

// ✅ GOOD - keep pointers as unsafe.Pointer for the duration of the call.
buf := make([]byte, 64)
p := unsafe.Pointer(&buf[0])
_, _ = ffi.CallFunction(&cif, fn, nil, []unsafe.Pointer{unsafe.Pointer(&p)})
runtime.KeepAlive(buf)

// ✅ If C keeps the pointer after the call returns, or a callback reads it
// later, pin the memory (runtime.Pinner) or allocate it on the C side.
```

C strings must be NUL-terminated (`"hello\x00"`); Go strings are not.

### 2. Race Conditions

**Risk**: FFI calls involve low-level system calls and assembly code. Concurrent access can cause race conditions.

**Example Attack**:
```
Thread 1: Prepares CallInterface with argTypes=[int32]
Thread 2: Modifies argTypes=[pointer] before call
Result: Type confusion, memory corruption
```

**Mitigation**:
- ✅ CI runs the test suite with `-race` on every runner that has a race detector
- ✅ A prepared `CallInterface` is only read by `CallFunction`; a `CallInterface` built by hand is classified on a copy, never written
- ⚠️ Users must ensure `CallInterface` is not modified during use

**User Best Practices**:
```go
// ❌ BAD - Preparing a CallInterface while another goroutine calls it
var globalCIF types.CallInterface
go func() {
    ffi.PrepareCallInterface(&globalCIF, ...) // Race!
}()
go func() {
    ffi.CallFunction(&globalCIF, ...) // Race!
}()

// ✅ GOOD - One CallInterface per function, prepared once
cif := &types.CallInterface{}
ffi.PrepareCallInterface(cif, ...)
// Safe to call concurrently after preparation (each goroutine with its own argument values)
go func() { ffi.CallFunction(cif, ...) }()
go func() { ffi.CallFunction(cif, ...) }()
```

### 3. Platform-Specific Assembly Bugs

**Risk**: goffi uses hand-written assembly for the System V AMD64 and AAPCS64 ABIs (Windows AMD64 calls go through Go's `syscall.SyscallN`). Assembly bugs can cause crashes or security vulnerabilities.

**Attack Vectors**:
- Stack corruption due to incorrect frame setup
- Register corruption if calling convention violated
- Red zone violations on System V (128 bytes below RSP)
- Shadow space violations on Win64 (32 bytes for 4 parameters)
- Incorrect float/SSE register handling

**Mitigation**:
- ✅ ABI rules taken from the ABI specifications and libffi
- ✅ Bugs are first reproduced by a failing test, then fixed
- ✅ Tests on native CI runners: Linux, Windows and macOS on amd64 and arm64; FreeBSD amd64 in VMs
- ✅ C-backed end-to-end tests (struct sizes and classes, stack arguments, narrow integers, variadic, callbacks from C)

**Current Status**: see the platform table in [README.md](README.md#platform-support). FreeBSD arm64 is cross-compiled only; Android arm64 is a guarded preview.

### 4. Type Safety Violations

**Risk**: FFI requires matching Go types to C types. Mismatches can cause crashes or undefined behavior.

**Example Attack**:
```go
// C function: void process(int32_t value)
// Go code:
var value uint64 = 0xFFFFFFFF00000001
ffi.CallFunction(&cif, funcPtr, nil, []unsafe.Pointer{
    unsafe.Pointer(&value), // ❌ Passing uint64 as int32!
})
// Result: C function reads garbage or crashes
```

**Mitigation**:
- ✅ Type descriptors are validated in `PrepareCallInterface`
- ✅ Typed errors with field context (`TypeValidationError`, `InvalidCallInterfaceError`, …)
- ⚠️ goffi cannot check that a descriptor matches the C prototype — that is the caller's job

**User Best Practices**:
```go
// ✅ ALWAYS match Go types to C types exactly
// C: void func(int32_t x, double y, const char* s)
var x int32 = 42
var y float64 = 3.14
var s = "hello\x00"

ffi.PrepareCallInterface(&cif, types.DefaultCall,
    types.VoidTypeDescriptor,
    []*types.TypeDescriptor{
        types.SInt32TypeDescriptor,    // int32
        types.DoubleTypeDescriptor,    // double
        types.PointerTypeDescriptor,   // const char*
    },
)
```

### 5. Dynamic Library Loading Risks

**Risk**: Loading arbitrary dynamic libraries can execute malicious code.

**Attack Vectors**:
- DLL hijacking on Windows (loading a malicious DLL found earlier in the search order)
- `LD_PRELOAD` / `LD_LIBRARY_PATH` attacks on Linux
- Path traversal in library names
- Loading libraries with constructor functions (immediate code execution)

**Mitigation**:
- ⚠️ User responsible for validating library paths
- ⚠️ goffi uses the OS loader (`LoadLibraryW` / `dlopen`) with its default search order — no custom path logic
- ✅ Names containing NUL bytes are rejected

**User Best Practices**:
```go
// ❌ BAD - Loading user-provided library paths
libPath := os.Args[1] // User input!
handle, _ := ffi.LoadLibrary(libPath) // Malicious DLL!

// ✅ GOOD - Validate and sanitize library names
allowedLibs := map[string]bool{
    "wgpu_native.dll": true,
    "libvulkan.so.1": true,
}
if !allowedLibs[libName] {
    return errors.New("library not allowed")
}
handle, err := ffi.LoadLibrary(libName)
```

### 6. Resource Exhaustion

**Risk**: Leaking library handles or excessive FFI calls can exhaust resources.

**Mitigation**:
- ✅ `FreeLibrary()` provided for cleanup
- ✅ Documentation emphasizes `defer ffi.FreeLibrary(handle)`
- ✅ Context support (`CallFunctionContext`) — checked before the call starts; a C function that is already running cannot be interrupted

**User Best Practices**:
```go
// ✅ ALWAYS use defer to free libraries
handle, err := ffi.LoadLibrary("mylib.dll")
if err != nil {
    return err
}
defer ffi.FreeLibrary(handle) // Prevent leaks

// ✅ Use context for timeouts
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()
_, err = ffi.CallFunctionContext(ctx, &cif, funcPtr, unsafe.Pointer(&result), args)
```

## Known Security Considerations

### 1. Memory Safety

**Status**: Actively mitigated via type validation and runtime checks.

**Risk Level**: Medium to High

**Description**: FFI inherently involves `unsafe.Pointer` operations. Incorrect pointer handling can lead to crashes or memory corruption.

**Mitigation**:
- Type descriptors validated before use
- Assembly code reviewed for stack safety
- Tests with the race detector and `checkptr`
- User documentation emphasizes safe pointer handling

### 2. Platform-Specific Bugs

**Status**: Tested on native CI runners for every production target except FreeBSD arm64.

**Risk Level**: Medium

**Description**: Hand-written assembly and per-platform argument layout may contain ABI compliance bugs.

**Mitigation**:
- C-backed end-to-end tests on Linux, Windows, macOS (amd64 and arm64) and FreeBSD amd64
- Each ABI fix is proven first by a test that fails on the unfixed code
- 🔄 Formal ABI verification tools (planned)

### 3. Dependency Security

goffi has **no module dependencies** — `go.mod` has no `require` lines:

- Pure Go implementation; with `CGO_ENABLED=0` no C code is compiled
- Standard library only (`unsafe`, `runtime`, `syscall`, `context`, …)
- Tests use only the standard `testing` package; the C-backed end-to-end tests compile `ffi/testdata/structtest.c` from source

## Security Testing

### Current Testing

- ✅ Unit tests with invalid arguments
- ✅ Race detector (with `checkptr`) on every CI runner that supports it
- ✅ golangci-lint (errcheck, govet, ineffassign, staticcheck, unused) on every target
- ✅ Native CI on Linux, Windows, macOS × amd64/arm64 and FreeBSD VMs
- ✅ Benchmark validation (FFI overhead 88-114ns)

### Planned for v1.0

- 🔄 Fuzzing
- 🔄 Static analysis with gosec and govulncheck in CI
- 🔄 Security audit by external experts
- 🔄 Formal ABI compliance verification

## Security Disclosure History

No security vulnerabilities have been reported or fixed yet.

When vulnerabilities are addressed, they will be listed here with:
- **CVE ID** (if assigned)
- **Affected versions**
- **Fixed in version**
- **Severity** (Critical/High/Medium/Low)
- **Credit** to reporter

## Security Contact

- **GitHub Security Advisory**: https://github.com/go-webgpu/goffi/security/advisories/new
- **Public Issues** (for non-sensitive bugs): https://github.com/go-webgpu/goffi/issues
- **Discussions**: https://github.com/go-webgpu/goffi/discussions

## Bug Bounty Program

goffi does not currently have a bug bounty program. We rely on responsible disclosure from the security community.

If you report a valid security vulnerability:
- ✅ Public credit in security advisory (if desired)
- ✅ Acknowledgment in CHANGELOG
- ✅ Priority review and quick fix

---

**Thank you for helping keep goffi secure!** 🔒

*Security is a journey, not a destination. We continuously improve our security posture with each release.*
