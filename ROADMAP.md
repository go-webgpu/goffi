# goffi - Development Roadmap

> **Strategic Approach**: Build production-ready Zero-CGO FFI with benchmarked performance
> **Philosophy**: Performance first, usability second, platform coverage third

**Last Updated**: 2026-02-18 | **Current Version**: v0.3.9 | **Strategy**: Benchmarks → Callbacks → ARM64 → Runtime → API → v1.0 LTS | **Milestone**: v0.3.9 (callback fixes) → v0.4.0 (crosscall2) → v0.5.0 Usability → v1.0.0 LTS

---

## 🎯 Vision

Build a **production-ready, zero-CGO FFI library for Go** with:
- **Performance**: < 200ns overhead (current: 88-114ns ✅)
- **Usability**: Simple, type-safe API
- **Coverage**: All major platforms and calling conventions
- **Quality**: 80%+ test coverage, comprehensive documentation

### Key Differentiators

✅ **Zero CGO Dependency**
- No C compiler required
- Easy cross-compilation
- Pure Go deployment

✅ **Performance First**
- Hand-optimized assembly for each ABI
- Benchmarked and validated
- < 5% overhead for GPU operations

✅ **Production Quality**
- 89.1% test coverage
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
v0.3.9 (CALLBACK FIXES) → 2026-02 (in progress)
         ↓ (runtime integration)
v0.4.0 (CROSSCALL2 INTEGRATION) → 2026 Q1-Q2
         ↓ (usability)
v0.5.0 (USABILITY + VARIADIC) → 2026 Q2-Q3
         ↓ (advanced features)
v0.8.0 (ADVANCED FEATURES) → 2026 Q3-Q4
         ↓ (community adoption + validation)
v1.0.0 LTS → Long-term support release (Q1 2026)
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

**v0.3.9** = Callback fixes (2026-02, in progress)
- **ARM64 callback trampoline rewrite** (BL→MOVD+B)
- **Symbol rename** to avoid purego linker collision ([#15](https://github.com/go-webgpu/goffi/issues/15))
- Package-scoped assembly symbols (`·callbackTrampoline`/`·callbackDispatcher`)

**v0.4.0** = Runtime integration (2026 Q1-Q2)
- **crosscall2 integration** for C-thread callbacks ([#16](https://github.com/go-webgpu/goffi/issues/16))
- Proper C→Go transition: `runtime·load_g` + `runtime·cgocallback`
- Support callbacks from arbitrary C threads (wgpu-native internal threads)

**v0.5.0** = Usability + Variadic (2026 Q2-Q3)
- Builder pattern API
- Platform-specific struct handling
- **Variadic function support** (printf, sprintf, etc.)

**v1.0.0** = Long-term support release (Q1 2026)
- API stability guarantee
- Security audit
- Published benchmarks vs CGO/purego
- 3+ years LTS support

---

## 📊 Current Status (v0.3.9)

**Phase**: Callback fixes + ARM64 trampoline rewrite

**What Works**:
- ✅ Dynamic library loading (`LoadLibrary`, `GetSymbol`, `FreeLibrary`)
- ✅ Function call interface (`PrepareCallInterface`)
- ✅ Function execution (`CallFunction`, `CallFunctionContext`)
- ✅ **Benchmarks**: 64-114 ns/op FFI overhead ✨
- ✅ **Typed errors**: 5 error types with `errors.As()` support
- ✅ **Context support**: Timeouts and cancellation
- ✅ **Cross-platform**: Linux + Windows + macOS (AMD64 + ARM64)
- ✅ **Type system**: Predefined descriptors for common types
- ✅ **Callbacks**: C-to-Go function calls (2000 entries)
- ✅ **89.6% test coverage** (exceeded 80% target)

**Performance**:
- ✅ AMD64: **88-114 ns/op** (Intel i7-1255U)
- ✅ ARM64: **64 ns/op** (Apple M3 Pro)
- ✅ **Verdict**: < 5% overhead for WebGPU operations (target achieved!)

**Platform Support**:
- ✅ Linux AMD64 (System V ABI)
- ✅ Windows AMD64 (Win64 ABI)
- ✅ macOS AMD64 (System V ABI)
- ✅ Linux ARM64 (AAPCS64 ABI)
- ✅ macOS ARM64 (AAPCS64 ABI) - Apple Silicon M1/M2/M3/M4

**Requirements**:
- ✅ `CGO_ENABLED=0` required (clear error message if CGO_ENABLED=1)
- ✅ Go 1.21+ recommended

**Documentation**:
- ✅ README.md with benchmarks and requirements
- ✅ docs/PERFORMANCE.md (comprehensive guide)
- ✅ CHANGELOG.md with migration guide
- ✅ CONTRIBUTING.md
- ✅ CODE_OF_CONDUCT.md
- ✅ SECURITY.md

---

## 📅 What's Next

### **v0.5.0 - Usability + Variadic** (Q2 2025)

**Goal**: Improve developer experience and add variadic function support

**Duration**: 2-3 months (Q2 2025)

**Features**:
1. **Builder Pattern API** (HIGH PRIORITY)
   ```go
   lib.Call("wgpuCreateInstance").
       Arg(nil).
       ReturnPointer(&instance)
   ```

2. **Variadic Function Support** (HIGH PRIORITY)
   - System V: AL register = SSE argument count
   - Win64: Float→GP register duplication
   - Examples: printf, sprintf, scanf

3. **Platform-Specific Struct Handling** (MEDIUM PRIORITY)
   - Windows `#pragma pack` support
   - MSVC vs GCC alignment differences

4. **Type-Safe Argument Helpers** (MEDIUM PRIORITY)
   ```go
   args := ffi.Args(ffi.Int32(42), ffi.String("hello"))
   ```

**Quality Targets**:
- Maintain 80%+ test coverage
- 0 linter issues
- API stability (no breaking changes)

---

### **v0.8.0 - Advanced Features** (Q4 2025)

**Goal**: Advanced FFI capabilities and tooling

**Duration**: 2-3 months (Q4 2025)

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

**Quality Targets**:
- Codegen tool with 90%+ C header coverage
- Tooling documentation
- Advanced examples (OpenGL, Vulkan)

---

### **v1.0.0 - Long-Term Support Release** (Q1 2026)

**Goal**: Production-ready LTS release with stability guarantees

**Requirements**:
- v0.8.x stable for 3+ months
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
- docs/PERFORMANCE.md - Performance guide
- CHANGELOG.md - Version history
- CONTRIBUTING.md - Development guide
- SECURITY.md - Security policy

**Development**:
- API_TODO.md - Detailed task backlog
- .github/workflows/ci.yml - CI/CD pipeline
- scripts/pre-release-check.sh - Quality checks

**Reference**:
- libffi: https://github.com/libffi/libffi
- purego: https://github.com/ebitengine/purego
- wgpu-native: https://github.com/gfx-rs/wgpu-native

---

## 📊 Quality Metrics

**Current (v0.3.9)**:
- ✅ Test coverage: 89.6% (target: 80%+)
- ✅ Linter issues: 0
- ✅ Benchmarks: 64-114 ns/op (AMD64 + ARM64)
- ✅ Platforms: 5 (Linux, Windows, macOS × AMD64/ARM64)
- ✅ CGO requirement: Clear error message

**Target (v1.0.0)**:
- 🎯 Test coverage: 90%+
- 🎯 Linter issues: 0
- 🎯 Benchmarks: < 100ns average
- 🎯 Platforms: 5+ (Linux, Windows, macOS × AMD64/ARM64)
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
- Comprehensive tests (unit + integration + benchmarks)
- Linting with 34+ security-focused linters
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
- Check API_TODO.md for open tasks
- Join discussions on roadmap priorities

**Security**:
- See SECURITY.md
- Private disclosure: https://github.com/go-webgpu/goffi/security/advisories/new

---

*Version 1.3 (Updated 2026-02-18)*
*Current: v0.3.9 (Callback fixes) | Next: v0.4.0 (crosscall2) | Target: v1.0.0 LTS*
