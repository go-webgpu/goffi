# goffi - Development Roadmap

> **Strategic Approach**: Build production-ready Zero-CGO FFI with benchmarked performance
> **Philosophy**: Performance first, usability second, platform coverage third

**Last Updated**: 2025-11-28 | **Current Version**: v0.2.1 | **Strategy**: Benchmarks → Callbacks → ARM64 → API → v1.0 LTS | **Milestone**: v0.2.1 RELEASED! → v0.3.0 ARM64 (Q1 2025) → v1.0.0 LTS (Q1 2026)

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
         ↓ (in progress - ARM64 implementation)
v0.3.0 (ARM64 SUPPORT) → Q1 2025
         ↓ (2-3 months)
v0.5.0 (USABILITY + VARIADIC) → Q2 2025
         ↓ (2-3 months)
v0.8.0 (ADVANCED FEATURES) → Q3 2025
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

**v0.3.0** = ARM64 support 🟡 IN DEVELOPMENT
- **ARM64 architecture support** (Linux + macOS AAPCS64 ABI)
- Cross-compile verified, awaiting real hardware testing
- Feature branch: `feature/arm64-support`
- Requested by: go-webgpu project (Apple Silicon support)

**v0.5.0** = Usability + Variadic (Q2 2025)
- Builder pattern API
- Platform-specific struct handling
- **Variadic function support** (printf, sprintf, etc.)

**v1.0.0** = Long-term support release (Q1 2026)
- API stability guarantee
- Security audit
- Published benchmarks vs CGO/purego
- 3+ years LTS support

---

## 📊 Current Status (v0.1.0)

**Phase**: ✅ Performance Validated + Production Ready

**What Works**:
- ✅ Dynamic library loading (`LoadLibrary`, `GetSymbol`, `FreeLibrary`)
- ✅ Function call interface (`PrepareCallInterface`)
- ✅ Function execution (`CallFunction`, `CallFunctionContext`)
- ✅ **Benchmarks**: 88-114 ns/op FFI overhead ✨
- ✅ **Typed errors**: 5 error types with `errors.As()` support
- ✅ **Context support**: Timeouts and cancellation
- ✅ **Cross-platform**: Linux + Windows AMD64
- ✅ **Type system**: Predefined descriptors for common types
- ✅ **89.1% test coverage** (exceeded 80% target)

**Performance**:
- ✅ BenchmarkGoffiOverhead: **88.09 ns/op** (empty function)
- ✅ BenchmarkGoffiIntArgs: **113.9 ns/op** (integer args)
- ✅ BenchmarkGoffiStringOutput: **97.81 ns/op** (string processing)
- ✅ BenchmarkDirectGo: **0.21 ns/op** (baseline)
- ✅ **Verdict**: < 5% overhead for WebGPU operations (target achieved!)

**Platform Support**:
- ✅ Linux AMD64 (System V ABI)
- ✅ Windows AMD64 (Win64 ABI)
- ✅ macOS AMD64 (System V ABI) - v0.1.1
- 🟡 ARM64 Linux/macOS (AAPCS64 ABI) - in development for v0.3.0

**Documentation**:
- ✅ README.md with real benchmarks
- ✅ docs/PERFORMANCE.md (comprehensive guide)
- ✅ CHANGELOG.md with migration guide
- ✅ CONTRIBUTING.md
- ✅ CODE_OF_CONDUCT.md
- ✅ SECURITY.md

---

## 📅 What's Next

### **v0.3.0 - ARM64 Support** (Q1 2025) 🟡 IN DEVELOPMENT

**Goal**: Full ARM64 platform support for Apple Silicon and Linux ARM64

**Status**: Cross-compile verified, awaiting real hardware testing

**Completed**:
- ✅ `internal/arch/arm64/` - Implementation, classification, call_unix
- ✅ `internal/syscall/arm64` - Call8Float wrapper and assembly
- ✅ `internal/dl/arm64` - Dynamic loader stubs and wrappers
- ✅ `ffi/callback_arm64` - 2000-entry trampoline table
- ✅ Cross-compile: `GOOS=linux GOARCH=arm64` builds
- ✅ Cross-compile: `GOOS=darwin GOARCH=arm64` builds (Apple Silicon)

**Pending**:
- [ ] Real ARM64 hardware testing (Linux ARM64 / macOS M1+)
- [ ] CI/CD ARM64 runners (GitHub Actions `macos-latest`)
- [ ] Performance benchmarks on ARM64
- [ ] Documentation updates

**ARM64 AAPCS64 ABI Implementation**:
- X0-X7: 8 integer/pointer registers
- D0-D7: 8 floating-point registers
- X8: Indirect result location
- Homogeneous Floating-point Aggregate (HFA) support
- 2000 callback trampolines

---

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

**Current (v0.1.0)**:
- ✅ Test coverage: 89.1% (target: 80%+)
- ✅ Linter issues: 0
- ✅ Benchmarks: 88-114 ns/op
- ✅ Platforms: 2 (Linux, Windows AMD64)

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

*Version 1.1 (Updated 2025-11-28)*
*Current: v0.2.1 (Callbacks + Windows Hotfix) | Phase: ARM64 Development | Next: v0.3.0 (ARM64) | Target: v1.0.0 LTS (Q1 2026)*
