# goffi - Development Roadmap

> **Strategic Approach**: Build production-ready Zero-CGO FFI with benchmarked performance
> **Philosophy**: Performance first, usability second, platform coverage third

**Last Updated**: 2026-10-18 | **Current Version**: v0.6.7 | **Strategy**: Benchmarks → Callbacks → ARM64 → Runtime → ABI → v1.0 LTS | **Next**: v0.6.8 (`CGO_ENABLED=1` link fix, [#72](https://github.com/go-webgpu/goffi/issues/72)) → v0.7.0 (ergonomic API) → v1.0.0 LTS. Live plan: [milestones](https://github.com/go-webgpu/goffi/milestones)

---

## 🎯 Vision

Build a **production-ready, zero-CGO FFI library for Go** with:
- **Performance**: < 200ns overhead (current: 88-114ns ✅)
- **Usability**: Simple, type-safe API
- **Coverage**: All major platforms and calling conventions
- **Quality**: high test coverage (CI gate 70%, ~90% on amd64), every ABI bug proven by a failing test before the fix

### Key Differentiators

✅ **Zero CGO Dependency**
- No C compiler required with `CGO_ENABLED=0` (`CGO_ENABLED=1` is supported too)
- Easy cross-compilation
- Pure Go deployment

✅ **Performance First**
- Hand-written assembly for System V AMD64 and AAPCS64; Win64 through Go's `syscall.SyscallN`
- Benchmarked and validated
- < 5% overhead for GPU operations

✅ **Production Quality**
- ~90% test coverage on amd64
- C-backed end-to-end ABI tests on native Linux, Windows and macOS runners (amd64 + arm64) and FreeBSD VMs
- Typed error system
- Comprehensive documentation
- Active maintenance

---

## 🚀 Version Strategy

### Philosophy: Performance → Usability → Coverage → Stable

```
v0.1.0 (BENCHMARKS + QUALITY) ✅ RELEASED 2025-11-17
         ↓ (1 day - macOS completion)
v0.1.1 (macOS SUPPORT) ✅ RELEASED 2025-11-18
         ↓ (9 days - callback implementation)
v0.2.0 (CALLBACKS) ✅ RELEASED 2025-11-27
         ↓ (1 day - Windows hotfix)
v0.2.1 (WINDOWS HOTFIX) ✅ RELEASED 2025-11-27
         ↓ (ARM64 implementation)
v0.3.0-v0.3.7 (ARM64 SUPPORT) ✅ RELEASED 2025-12-29
         ↓ (CGO error handling)
v0.3.8 (CGO ERROR HANDLING) ✅ RELEASED 2026-01-24
         ↓ (callback fixes)
v0.3.9 (CALLBACK FIXES) ✅ RELEASED 2026-02-18
         ↓ (runtime integration)
v0.4.0 (CROSSCALL2 INTEGRATION) ✅ RELEASED 2026-02-27
         ↓ (usability)
v0.5.0 (PLATFORM COVERAGE) ✅ RELEASED 2026-03-29
         ↓ (struct ABI + CGO support)
v0.5.1 (STRUCT ABI + CGO_ENABLED=1) ✅ RELEASED 2026-05-13
         ↓ (variadic + vet fixes)
v0.5.2 (VARIADIC FUNCTIONS) ✅ RELEASED 2026-05-25
         ↓ (FreeBSD arm64, HostLayout, callback stack-move fix)
v0.5.3-v0.5.6 ✅ RELEASED 2026-05-28 … 2026-07-05
         ↓ (errno)
v0.6.0 (ERRNO ALWAYS-CAPTURE, BREAKING) ✅ RELEASED 2026-07-12
         ↓ (Android preview, Windows floats, static profile, ABI fixes)
v0.6.1-v0.6.7 ✅ RELEASED 2026-07-21 … 2026-10-18
         ↓ (CGO_ENABLED=1 link fix)
v0.6.8 (PATCH) → next
         ↓ (ergonomic API, arm32, host profiles)
v0.7.0 (ERGONOMIC API) → next minor
         ↓ (community adoption + validation)
v1.0.0 LTS → Long-term support release (2027 Q1)
```

### Critical Milestones

**v0.1.0** = Performance validated, production-ready for WebGPU ✅ RELEASED (2025-11-17)
- **88-114 ns/op** FFI overhead (BETTER than estimated 230ns!)
- 89.1% test coverage
- 5 typed errors
- Platform: Linux + Windows AMD64

**v0.1.1** = macOS support completion ✅ RELEASED (2025-11-18)
- **macOS AMD64** added to supported platforms
- System V ABI shared implementation (Linux + macOS)
- CI/CD coverage: 3 platforms (Linux, Windows, macOS)
- Coverage: **87.1%** (accurate calculation)

**v0.2.0** = Callback support for async APIs ✅ RELEASED (2025-11-27)
- **NewCallback API** for C-to-Go function calls
- 2000-entry trampoline table
- Thread-safe callback registry
- WebGPU async operations now supported
- Requested by: go-webgpu/webgpu project

**v0.2.1** = Windows callback hotfix ✅ RELEASED (2025-11-27)
- Windows ABI fix using `syscall.NewCallback`
- SEH exception limitation documented
- Platform-specific callback implementations

**v0.3.0-v0.3.7** = ARM64 support ✅ RELEASED (2025-12-29)
- **ARM64 architecture support** (Linux + macOS AAPCS64 ABI)
- Tested on Apple Silicon M3 Pro (64 ns/op)
- HFA returns, nested structs, mixed int/float support
- Contributed by: @ppoage (PR #9)

**v0.3.8** = CGO error handling ✅ RELEASED (2026-01-24)
- **Enterprise-grade CGO_ENABLED=1 error handling**
- Compile-time assertion: `GOFFI_REQUIRES_CGO_ENABLED_0`
- Clear documentation in README.md Requirements section
- Fixes confusing linker errors on Linux/macOS with gcc/clang

**v0.3.9** = Callback fixes ✅ RELEASED (2026-02-18)
- **ARM64 callback trampoline rewrite** (BL→MOVD+B)
- **Symbol rename** to avoid purego linker collision ([#15](https://github.com/go-webgpu/goffi/issues/15))
- Package-scoped assembly symbols (`·callbackTrampoline`/`·callbackDispatcher`)

**v0.4.0** = Runtime integration ✅ RELEASED (2026-02-27)
- **crosscall2 integration** for C-thread callbacks ([#16](https://github.com/go-webgpu/goffi/issues/16))
- Proper C→Go transition: `crosscall2 → runtime·load_g → runtime·cgocallback`
- Support callbacks from arbitrary C threads (Metal, wgpu-native internal threads)
- fakecgo trampoline register fixes

**v0.4.1** = ABI compliance hotfix ✅ RELEASED (2026-03-02)
- **Full ABI compliance audit** — 10 of 11 gaps fixed ([#19](https://github.com/go-webgpu/goffi/issues/19))
- AMD64/ARM64 stack spill for arguments beyond register count
- Float32 argument encoding fix (`math.Float32bits`)
- Struct return 9-16 bytes, sret hidden pointer, HFA stack spill
- Overflow detection, `runtime.KeepAlive` safety

**v0.5.0** = Platform coverage ✅ RELEASED (2026-03-29)
- **Windows ARM64** support (Snapdragon X Elite, tested by @SideFx)
- **FreeBSD amd64** support (cross-compile verified)
- 7 platform targets (Linux/Windows/macOS/FreeBSD × amd64 + Linux/macOS/Windows ARM64)

**v0.5.1** = Struct ABI + CGO_ENABLED=1 ✅ RELEASED (2026-05-13)
- **CGO_ENABLED=1 support** (PR #37 by @jiyeyuran) — dual-mode build, race detector compatible
- **Struct by-value argument passing** (PR #39, closes #33) — ≤8B/9-16B/>16B, INTEGER/SSE classification
- **Callback struct arguments** (PR #42 by @pekim, closes #41) — C→Go callbacks with struct args
- **9-16B struct return via XMM** (TASK-045) — 4 return modes: RAX:RDX, RAX:XMM0, XMM0:RAX, XMM0:XMM1
- **Race detector** — checkptr double-indirection fix (Go #58625), `-race` clean
- **E2E test infrastructure** — gcc-compiled C test library for struct passing verification
- Contributors: @jiyeyuran (CGO path maintainer), @pekim (callback structs)

**v0.5.2** = Variadic functions ✅ RELEASED (2026-05-25)
- **Variadic function support** — `PrepareVariadicCallInterface` with Apple ARM64 stack-force

**v0.5.3** = FreeBSD ARM64 ✅ RELEASED (2026-05-28)
- Build tag fix for FreeBSD ARM64 (8 platforms total)

**v0.5.4** = structs.HostLayout ✅ RELEASED (2026-06-15)
- ABI-safe struct layout for all assembly-interface structures

**v0.5.5** = Example fix + CI ✅ RELEASED (2026-06-15)
- Example avalue pointer bug fix, CI examples build verification

**v0.5.6** = Callback stack-move fix ✅ RELEASED (2026-07-05)
- Critical: `syscallArgs` moved to sync.Pool (goroutine stack-move safety)
- Discovered by @tie — `TestCallbackGrowStack` reproducer

**v0.6.0** = errno always-capture ✅ RELEASED (2026-07-12)
- **BREAKING**: `CallFunction` returns `(syscall.Errno, error)` — always captures C errno
- First pure-Go FFI with correct errno capture on Linux
- Assembly-level capture inside trampoline (thread-safe window)
- Platform support: `__errno_location` (Linux), `__error` (macOS/FreeBSD)

**v0.6.1** = Android ARM64 + fakecgo cleanup ✅ RELEASED (2026-07-21)
- **Android ARM64 Bionic** (guarded preview) — PR #62 by @besmpl
- fakecgo `purego_` → `goffi_` rename + dual copyright — PR #63
- LICENSE + NOTICE updated to GoGPU ecosystem pattern
- @besmpl added as CODEOWNER for Android paths

**v0.6.2** = Windows float returns fix ✅ RELEASED (2026-07-22)
- Windows XMM0 float return capture (TASK-019 resolved) — PR #65 by @besmpl

**v0.6.3** = ARM64 HFA checkptr fix ✅ RELEASED (2026-08-01)
- ARM64 `handleHFAReturn` checkptr crash fix (#67, reported by @jbunds)
- ARM64 9-16B struct return proactive fix (copy pattern)

**v0.6.4** = Static linking profile + struct examples ✅ RELEASED (2026-09-10)
- `-tags goffi_static` fully static Linux ELFs (#74, gogpu#474) — PR #78
- Linking-mode docs + ELF CI gates (`scripts/check-elf-linking.sh`)
- ADR-001 userspace ELF loader (research)
- Struct pass/return examples (`examples/struct/`, #58 / PR #69)
- Docs: avalue out-pointer double-indirection (#79 / PR #80)

**v0.6.5** = fakecgo fixes + native CI on every target ✅ RELEASED (2026-10-16)
- macOS/FreeBSD signal mask on new threads (per-OS `SIG_*`) — PR #90
- Linux `syscall.Setuid` & co. no longer panic; thread-exit crashes on amd64 (Go 1.28 `LEAVE`) and linux/arm64 fixed; FreeBSD `libthr.so.3`; NetBSD fakecgo layouts — PR #92
- FreeBSD amd64 callback trampolines link as code (#83 by @unxed)
- NUL bytes in `LoadLibrary` / `GetSymbol` names rejected
- CI: tests run natively on Linux/Windows/macOS × amd64/arm64 on Go 1.25 + stable, lint on every target

**v0.6.6** = ABI fixes ✅ RELEASED (2026-10-16)
- `float32` callback arguments and float return values on Unix amd64/arm64 ([#93](https://github.com/go-webgpu/goffi/issues/93))
- arm64 argument layout per AAPCS64 and Apple's arm64 ABI: mixed structs, HFA and small-struct stack overflow, darwin stack packing, no silently dropped arguments ([#96](https://github.com/go-webgpu/goffi/issues/96), [#97](https://github.com/go-webgpu/goffi/issues/97))
- `IntTypeDescriptor` passed and returned by value ([#99](https://github.com/go-webgpu/goffi/issues/99)); signed narrow integers sign-extended on amd64 System V (clang-built callees)
- `CallFunction` requires exactly one argument value per argument type
- C-backed end-to-end tests fail instead of skipping when the C test library cannot be built

**v0.6.7** = Struct ABI fixes + race detector and FreeBSD in CI ✅ RELEASED (2026-10-18)
- Struct results of a `types.CallInterface` filled without `PrepareCallInterface` classified on each call
- Windows amd64: structs of sizes other than 1, 2, 4 and 8 bytes returned through the hidden result pointer and passed as a pointer to a copy
- Struct results smaller than 8 bytes written with their exact size; System V struct arguments read with their exact size; one-eightbyte float structs (System V) and single-`float`/`double` structs (arm64) read from the floating-point register
- `LoadLibrary` / `GetSymbol` errors read on the thread whose `dlopen` / `dlsym` failed
- CI: `go test -race` on every cgo runner that supports it; the test suite runs on FreeBSD 14 and 15 in a VM
- MIT license headers on original files, checked in CI

**v0.6.8** = Patch (planned, [milestone](https://github.com/go-webgpu/goffi/milestone/9))
- Linux and FreeBSD: `CGO_ENABLED=1` binaries that also contain other cgo code, or are linked externally (`-linkmode=external`, `c-shared`, `c-archive`), fail to link ([#72](https://github.com/go-webgpu/goffi/issues/72)). The approach is discussed in the RFC [#104](https://github.com/go-webgpu/goffi/issues/104) before anything is implemented
- Further fixes accumulate here

**v0.7.0** = Ergonomic API (planned, [milestone](https://github.com/go-webgpu/goffi/milestone/7))
- RegisterFunc convenience API
- Library struct + OpenLibraryBytes
- NewFunc/Call/CallCtx ergonomic wrappers
- Enterprise C-ABI / Rust-cdylib host profile — guide, example and CI ([#81](https://github.com/go-webgpu/goffi/issues/81))
- Static single binary with FFI ([#74](https://github.com/go-webgpu/goffi/issues/74))
- linux/arm (GOARCH=arm) ([#70](https://github.com/go-webgpu/goffi/issues/70))
- libvips bindings as a real-world stress test ([#71](https://github.com/go-webgpu/goffi/issues/71)) — see below

**libvips stress-test target** ([#71](https://github.com/go-webgpu/goffi/issues/71), idea by @unxed). A binding of this size exercises what goffi needs for large C APIs:
- float arguments and results in callbacks on Unix — done ([#93](https://github.com/go-webgpu/goffi/issues/93), v0.6.6);
- struct-by-value callback arguments on arm64 — not implemented yet (amd64 Unix only today);
- NULL-terminated varargs (`vips_image_new_from_file`, `vips_call`) — `PrepareVariadicCallInterface` needs one CIF per argument-type combination, so ergonomics matter;
- RegisterFunc / ergonomic API (this release) — makes a binding of this size practical.
GObject (GValue, the type system, reference counting) belongs to the binding layer, not to goffi.

**v1.0.0** = Long-term support release (2027 Q1)
- API stability guarantee
- Security audit
- Published benchmarks vs CGO/purego
- 3+ years LTS support

---

## 📊 Current Status (v0.6.7)

**Phase**: struct ABI fixes shipped (hand-built `CallInterface`, Win64 hidden result pointer and by-reference copies, exact-size small structs), `dlerror` read on the right thread, race detector and FreeBSD VMs in CI. Each bug was proven by a failing test first. Next: the `CGO_ENABLED=1` link fix (v0.6.8, after the RFC #104 discussion), then the ergonomic API for v0.7.0

**What Works**:
- ✅ Dynamic library loading (`LoadLibrary`, `GetSymbol`, `FreeLibrary`)
- ✅ Function call interface (`PrepareCallInterface`, `PrepareVariadicCallInterface`)
- ✅ Function execution (`CallFunction`, `CallFunctionContext`) with errno always captured
- ✅ **Structs** by value as arguments and results on every ABI
- ✅ **`-tags goffi_static`** — fully static Linux ELFs (FFI unavailable; `errors.Is(err, ErrStaticBuild)`)
- ✅ **Benchmarks**: 88-114 ns/op FFI overhead (AMD64)
- ✅ **Typed errors**: 5 error types with `errors.As()` support (+ `ErrStaticBuild` sentinel)
- ✅ **Context support**: Timeouts and cancellation
- ✅ **Cross-platform**: Linux, Windows, macOS, FreeBSD × AMD64 + ARM64; Android arm64 preview
- ✅ **Type system**: Predefined descriptors for common types
- ✅ **Callbacks**: C-to-Go function calls (2000 entries on Unix; `syscall.NewCallback` on Windows)
- ✅ **Test coverage**: ~90% on amd64 (CI gate: 70%)

**Performance**:
- ✅ AMD64: **88-114 ns/op** (Intel i7-1255U)
- ✅ ARM64: **64 ns/op** (Apple M3 Pro)
- ✅ **Verdict**: < 5% overhead for WebGPU operations (target achieved!)

**Platform Support**: see the table in [README.md](README.md#platform-support) — how each target is verified (native CI runner, FreeBSD VM, cross-compile, preview).

**Requirements**:
- ✅ Go 1.25+ (the `go` line in `go.mod`)
- ✅ `CGO_ENABLED=0` (recommended, no C compiler) or `CGO_ENABLED=1` — on Linux and FreeBSD the latter does not yet link together with other cgo code ([#72](https://github.com/go-webgpu/goffi/issues/72))

**Documentation**:
- ✅ README.md with benchmarks and requirements
- ✅ docs/ARCHITECTURE.md, docs/PERFORMANCE.md, docs/ANDROID.md
- ✅ CHANGELOG.md with migration guide
- ✅ CONTRIBUTING.md
- ✅ CODE_OF_CONDUCT.md
- ✅ SECURITY.md

---

## 📅 What's Next

The two planned releases are described above (**v0.6.8**, **v0.7.0**); their current scope is in the [milestones](https://github.com/go-webgpu/goffi/milestones). Ideas without a release yet are tracked in the Backlog milestone.

### Ideas — not scheduled

**Usability**:
1. **Builder-style call API**
   ```go
   lib.Call("wgpuCreateInstance").
       Arg(nil).
       ReturnPointer(&instance)
   ```
2. **Type-Safe Argument Helpers**
   ```go
   args := ffi.Args(ffi.Int32(42), ffi.String("hello"))
   ```
3. **Platform-Specific Struct Handling** — Windows `#pragma pack`, MSVC vs GCC alignment differences

**Advanced Features**:
1. **Codegen Tool (`goffi-gen`)** (HIGH PRIORITY)
   ```bash
   goffi-gen --input=wgpu.h --output=wgpu.go
   ```
   - C header parser
   - Go binding generator
   - JSON intermediate format
   - Type mapping customization

2. **Struct Builder API** (MEDIUM PRIORITY)
   ```go
   structType := ffi.Struct().
       Field("x", types.Float32).
       Field("y", types.Float32).
       Build()
   ```

3. **Performance Optimizations** (MEDIUM PRIORITY)
   - JIT stub generation (reduce indirect jumps)
   - Batch API (`CallBatch()` for multiple calls)
   - Assembly micro-optimizations (target: 70ns)

4. **Thread-Local Storage (TLS)** (LOW PRIORITY)
   - Per-thread state management
   - OpenGL context binding support
   - Thread-safe library handles

5. **Userspace ELF loader** (research) — load `.so` files without host `dlopen`, for static binaries ([ADR-001](docs/ADR-001-userspace-elf-loader.md))

---

### **v1.0.0 - Long-Term Support Release** (2027 Q1)

**Goal**: Production-ready LTS release with stability guarantees

**Requirements**:
- The last 0.x release line stable for 3+ months
- Positive community feedback
- No critical bugs
- API proven in production (WebGPU, Vulkan, etc.)

**LTS Guarantees**:
- ✅ **API stability** (no breaking changes in v1.x.x)
- ✅ **Long-term support** (3+ years)
- ✅ **Semantic versioning** strictly followed
- ✅ **Security updates** and bug fixes
- ✅ **Performance improvements** (non-breaking)
- ✅ **Documentation** maintained and updated

**Deliverables**:
1. **Security Audit** by external experts
2. **Benchmark Suite** vs CGO/purego (published)
3. **Reference Implementations**:
   - WebGPU bindings (wgpu-native)
   - Vulkan bindings
   - SQLite bindings
4. **Comprehensive Documentation**:
   - Book-style guide
   - API reference (pkg.go.dev)
   - Video tutorials
5. **Support Policy**:
   - GitHub Discussions for Q&A
   - Issue triage within 48h
   - Critical fixes within 1 week

---

## 📚 Resources

**Official Documentation**:
- README.md - Project overview
- docs/ARCHITECTURE.md - Technical architecture
- docs/PERFORMANCE.md - Performance guide
- docs/ANDROID.md - Android arm64 preview
- CHANGELOG.md - Version history
- CONTRIBUTING.md - Development guide
- SECURITY.md - Security policy

**Development**:
- [Milestones](https://github.com/go-webgpu/goffi/milestones) and [issues](https://github.com/go-webgpu/goffi/issues) - Planned work
- .github/workflows/ci.yml - CI/CD pipeline
- scripts/pre-release-check.sh - Quality checks

**Reference**:
- libffi: https://github.com/libffi/libffi
- purego: https://github.com/ebitengine/purego
- wgpu-native: https://github.com/gfx-rs/wgpu-native

---

## 📊 Quality Metrics

**Current (v0.6.7)**:
- ✅ Test coverage: 90.9% linux/amd64, 89.5% windows/amd64, 83.4% arm64 (CI, 2026-10-18; gate 70%)
- ✅ Linter issues: 0 on every target
- ✅ Benchmarks: 64-114 ns/op (AMD64 + ARM64)
- ✅ Platforms: 8 desktop targets (Linux, Windows, macOS, FreeBSD × AMD64/ARM64) + Android arm64 preview
- ✅ Race detector in CI on every runner that supports it

**Target (v1.0.0)**:
- 🎯 Test coverage: 90%+
- 🎯 Linter issues: 0
- 🎯 Benchmarks: < 100ns average
- 🎯 Platforms: all 8 desktop targets tested natively (FreeBSD arm64 included)
- 🎯 Documentation: Comprehensive (book-style)
- 🎯 Community: Active (10+ contributors)

---

## 🔬 Development Philosophy

**Performance First**:
- Every change benchmarked
- Assembly optimized per platform
- Zero allocations in hot paths
- Profiling before optimization

**Quality Over Speed**:
- Comprehensive tests (unit + C-backed end-to-end + benchmarks)
- golangci-lint on every target
- Documentation updated with code
- Security-first design

**Community Driven**:
- Public roadmap (this file)
- Open issue discussion
- RFC process for major changes
- Contributor recognition

---

## 📞 Support & Feedback

**Questions**:
- GitHub Discussions: https://github.com/go-webgpu/goffi/discussions
- GitHub Issues: https://github.com/go-webgpu/goffi/issues

**Contributing**:
- See CONTRIBUTING.md
- Check the [milestones](https://github.com/go-webgpu/goffi/milestones) and open issues
- Join discussions on roadmap priorities

**Security**:
- See SECURITY.md
- Private disclosure: https://github.com/go-webgpu/goffi/security/advisories/new

---

*Version 1.5 (Updated 2026-10-18)*
*Current: v0.6.7 (struct ABI fixes) | Next: v0.6.8 (patch), v0.7.0 (ergonomic API) | Target: v1.0.0 LTS*
