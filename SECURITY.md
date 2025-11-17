# Security Policy

## Supported Versions

goffi is currently in early development (v0.1.x). We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1.0 | :x:                |

Future stable releases (v1.0+) will follow semantic versioning with LTS support.

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in goffi, please report it responsibly.

### How to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report security issues by:

1. **Private Security Advisory** (preferred):
   https://github.com/go-webgpu/goffi/security/advisories/new

2. **Email** to maintainers:
   Create a private GitHub issue or contact via discussions

### What to Include

Please include the following information in your report:

- **Description** of the vulnerability
- **Steps to reproduce** the issue (include minimal code example if possible)
- **Affected versions** (which versions are impacted)
- **Potential impact** (memory corruption, race condition, crash, etc.)
- **Suggested fix** (if you have one)
- **Your contact information** (for follow-up questions)

### Response Timeline

- **Initial Response**: Within 48-72 hours
- **Triage & Assessment**: Within 1 week
- **Fix & Disclosure**: Coordinated with reporter

We aim to:
1. Acknowledge receipt within 72 hours
2. Provide an initial assessment within 1 week
3. Work with you on a coordinated disclosure timeline
4. Credit you in the security advisory (unless you prefer to remain anonymous)

## Security Considerations for FFI Libraries

goffi provides Foreign Function Interface (FFI) to C libraries without CGO. This introduces unique security risks related to low-level system calls and memory management.

### 1. Memory Safety Risks

**Risk**: FFI involves passing Go pointers to C code and vice versa, which can lead to memory corruption if handled incorrectly.

**Attack Vectors**:
- Invalid pointer addresses passed to C functions
- Buffer overflows when C code writes beyond Go-allocated memory
- Use-after-free when Go garbage collector moves memory
- Dangling pointers when C library frees memory unexpectedly
- Type confusion between Go and C representations

**Mitigation in Library**:
- ✅ Type descriptor validation before calls
- ✅ Bounds checking on argument counts
- ✅ Safe `runtime.cgocall` for GC coordination
- ✅ Assembly code thoroughly reviewed
- 🔄 Fuzzing for memory safety (planned for v0.5.0)

**User Recommendations**:
```go
// ❌ BAD - Passing Go slice directly (GC may move memory)
data := []byte("hello")
ffi.CallFunction(&cif, funcPtr, nil, []unsafe.Pointer{
    unsafe.Pointer(&data[0]), // ❌ May become invalid!
})

// ✅ GOOD - Pin memory or use C-allocated buffers
data := []byte("hello")
dataPtr := unsafe.Pointer(unsafe.StringData("hello\x00"))
ffi.CallFunction(&cif, funcPtr, nil, []unsafe.Pointer{dataPtr})
```

### 2. Race Conditions

**Risk**: FFI calls involve low-level system calls and assembly code. Concurrent access can cause race conditions.

**Example Attack**:
```
Thread 1: Prepares CallInterface with argTypes=[int32]
Thread 2: Modifies argTypes=[pointer] before call
Result: Type confusion, memory corruption
```

**Mitigation**:
- ✅ All tests run with `-race` detector
- ✅ `runtime.cgocall` provides some synchronization
- ⚠️ Users must ensure `CallInterface` is not modified during use
- 🔄 Thread-safety documentation (v0.2.0)

**User Best Practices**:
```go
// ❌ BAD - Sharing CallInterface across goroutines
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
// Safe to call concurrently after preparation
go func() { ffi.CallFunction(cif, ...) }()
go func() { ffi.CallFunction(cif, ...) }()
```

### 3. Platform-Specific Assembly Bugs

**Risk**: goffi uses hand-written assembly for System V (Linux) and Win64 (Windows) ABIs. Assembly bugs can cause crashes or security vulnerabilities.

**Attack Vectors**:
- Stack corruption due to incorrect frame setup
- Register corruption if calling convention violated
- Red zone violations on System V (128 bytes below RSP)
- Shadow space violations on Win64 (32 bytes for 4 parameters)
- Incorrect float/SSE register handling

**Mitigation**:
- ✅ Assembly code thoroughly reviewed by experts
- ✅ Comprehensive tests on both platforms
- ✅ Benchmark validation (FFI overhead within expected range)
- ✅ Platform-specific CI/CD testing
- 🔄 Formal ABI compliance verification (v0.5.0)

**Current Status**:
- Linux AMD64: ✅ Fully tested, production-ready
- Windows AMD64: ✅ Fully tested, production-ready
- macOS AMD64: 🔄 Planned for v0.5.0
- ARM64: 🔄 Planned for v0.5.0

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
- ✅ Type descriptors validate size and alignment
- ✅ Runtime type checking in `PrepareCallInterface`
- ✅ Comprehensive error messages with field context
- 🔄 Enhanced type validation (v0.2.0)

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
- DLL hijacking on Windows (loading malicious DLL from current directory)
- LD_PRELOAD attacks on Linux
- Path traversal in library names
- Loading libraries with constructor functions (immediate code execution)

**Mitigation**:
- ⚠️ User responsible for validating library paths
- ⚠️ goffi uses OS-provided `LoadLibrary`/`dlopen` (no custom logic)
- 🔄 Path validation helpers (v0.2.0)

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
- ✅ Context support for timeouts (`CallFunctionContext`)

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
err := ffi.CallFunctionContext(ctx, &cif, funcPtr, &result, args)
```

## Known Security Considerations

### 1. Memory Safety

**Status**: Actively mitigated via type validation and runtime checks.

**Risk Level**: Medium to High

**Description**: FFI inherently involves `unsafe.Pointer` operations. Incorrect pointer handling can lead to crashes or memory corruption.

**Mitigation**:
- Type descriptors validated before use
- Assembly code reviewed for stack safety
- Comprehensive testing with race detector
- User documentation emphasizes safe pointer handling

### 2. Platform-Specific Bugs

**Status**: Active testing on Linux and Windows.

**Risk Level**: Medium

**Description**: Hand-written assembly for each platform may contain ABI compliance bugs.

**Mitigation**:
- Both platforms thoroughly tested
- CI/CD runs tests on Linux and Windows
- Benchmark validation ensures correctness
- 🔄 **TODO (v0.5.0)**: Formal ABI verification tools

### 3. Variadic Functions NOT Supported

**Status**: Known limitation (not a vulnerability).

**Risk Level**: Low

**Description**: Variadic functions (`printf`, `sprintf`) are NOT supported. Attempting to use them may cause undefined behavior.

**Mitigation**:
- Clearly documented in README and CHANGELOG
- Use non-variadic wrappers (e.g., `puts` instead of `printf`)
- Planned support in v0.5.0

### 4. Dependency Security

goffi has **zero runtime dependencies**:

- Pure Go implementation (no CGO)
- No external libraries required
- Standard library only (`unsafe`, `runtime`, `syscall`, `context`)

**Testing dependencies** (not in production):
- None (uses standard `testing` package)

**Monitoring**:
- 🔄 Dependabot enabled (when repository goes public)
- ✅ Zero external attack surface

## Security Testing

### Current Testing

- ✅ Unit tests with invalid arguments
- ✅ Race detector on all tests (`-race`)
- ✅ Linting with golangci-lint (34+ linters)
- ✅ Platform-specific CI/CD (Linux + Windows)
- ✅ Benchmark validation (FFI overhead 88-114ns)

### Planned for v1.0

- 🔄 Fuzzing with go-fuzz or libFuzzer
- 🔄 Static analysis with gosec
- 🔄 SAST/DAST scanning in CI
- 🔄 Security audit by external experts
- 🔄 Formal ABI compliance verification

## Security Disclosure History

No security vulnerabilities have been reported or fixed yet (project is in early development).

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
- ✅ Our gratitude and recognition in README
- ✅ Priority review and quick fix

---

**Thank you for helping keep goffi secure!** 🔒

*Security is a journey, not a destination. We continuously improve our security posture with each release.*
