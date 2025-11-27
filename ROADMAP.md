# goffi - Development Roadmap

> **Strategic Approach**: Build production-ready Zero-CGO FFI with benchmarked performance
> **Philosophy**: Performance first, usability second, platform coverage third

**Last Updated**: 2025-11-27 | **Current Version**: v0.2.0 | **Strategy**: Benchmarks → Callbacks → API → Platforms → v1.0 LTS | **Milestone**: v0.2.0 RELEASED! → v0.3.0 (Q2 2025) → v1.0.0 LTS (Q1 2026)

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
         ↓ (3-4 months)
v0.3.0 (USABILITY) → Q2 2025
         ↓ (2-3 months)
v0.5.0 (PLATFORM EXPANSION) → Q3 2025
         ↓ (2-3 months)
v0.8.0 (ADVANCED FEATURES) → Q4 2025
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

**v0.3.0** = Developer experience improvements (Q2 2025)
- Builder pattern API
- Platform-specific struct handling
- Enhanced documentation
- More examples

**v0.5.0** = Platform expansion (Q3 2025)
- ARM64 support (Linux + macOS)
- Variadic functions

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
- ⏳ macOS AMD64 (planned v0.5.0)
- ⏳ ARM64 (planned v0.5.0)

**Documentation**:
- ✅ README.md with real benchmarks
- ✅ docs/PERFORMANCE.md (comprehensive guide)
- ✅ CHANGELOG.md with migration guide
- ✅ CONTRIBUTING.md
- ✅ CODE_OF_CONDUCT.md
- ✅ SECURITY.md

---

## 📅 What's Next

### **v0.2.0 - Usability Improvements** (Q2 2025)

**Goal**: Make FFI easier to use without sacrificing performance

**Duration**: 3-4 months (Q2 2025)

**Critical Features**:
1. **Builder Pattern API** (HIGH PRIORITY)
   ```go
   // Current (verbose)
   cif := &types.CallInterface{}
   ffi.PrepareCallInterface(cif, types.DefaultCall, returnType, argTypes)
   ffi.CallFunction(cif, funcPtr, &result, args)

   // Future (fluent)
   lib.Call("wgpuCreateInstance").
       Arg(nil).
       ReturnPointer(&instance)
   ```

2. **Platform-Specific Struct Handling** (HIGH PRIORITY)
   - Windows `#pragma pack` support
   - MSVC vs GCC alignment differences
   - Automatic platform detection
   - Manual override options

3. **Type-Safe Argument Helpers** (MEDIUM PRIORITY)
   ```go
   // Current
   arg := int32(42)
   args := []unsafe.Pointer{unsafe.Pointer(&arg)}

   // Future
   args := ffi.Args(ffi.Int32(42), ffi.String("hello"))
   ```

4. **Enhanced Documentation** (MEDIUM PRIORITY)
   - API reference (pkg.go.dev)
   - Tutorial series
   - 15+ real-world examples
   - Video guides (YouTube)

5. **Performance Profiling Tools** (LOW PRIORITY)
   ```go
   ffi.EnableProfiling() // Track call frequency and duration
   stats := ffi.GetStatistics() // Hotspot detection
   ```

**Quality Targets**:
- Maintain 80%+ test coverage
- 0 linter issues
- API stability (no breaking changes after v0.2.0)

---

### **v0.5.0 - Platform Expansion** (Q3 2025)

**Goal**: Support all major platforms and ABIs

**Duration**: 2-3 months (Q3 2025)

**Platform Features**:
1. **ARM64 Support** (CRITICAL)
   - Linux ARM64 (AAPCS64 ABI)
   - macOS Apple Silicon (AAPCS64 + Apple extensions)
   - Assembly implementation for ARM64
   - CI/CD on ARM64 runners

2. **macOS AMD64 Validation** (HIGH PRIORITY)
   - Validate System V ABI on macOS
   - Test with macOS system libraries
   - CI/CD on macOS runners

3. **Variadic Function Support** (HIGH PRIORITY)
   - System V: AL register = SSE argument count
   - Win64: Float→GP register duplication
   - Examples: printf, sprintf, scanf
   - Type-safe variadic helpers

4. **Callback Support** (MEDIUM PRIORITY)
   - C→Go function calls
   - Trampoline generation
   - Thread-safe callback registry
   - Example: GUI event handlers

5. **Windows ARM64** (LOW PRIORITY)
   - Experimental support
   - Windows ARM64 ABI
   - Limited testing (no CI yet)

**Quality Targets**:
- All platforms CI/CD tested
- Benchmarks on all platforms
- Cross-platform examples

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

*Version 1.0 (Updated 2025-01-17)*
*Current: v0.1.0 (Performance Validated) | Phase: Production Ready | Next: v0.2.0 (Usability) | Target: v1.0.0 LTS (Q1 2026)*
