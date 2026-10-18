# Contributing to goffi - Zero-CGO FFI for Go

Thank you for considering contributing to goffi! This document outlines the development workflow and guidelines.

## Git Workflow (GitHub Flow)

goffi uses GitHub Flow: one long-lived branch, `main`, and short-lived branches that reach it through pull requests.

- **main** is always releasable. It is protected: changes land only through a pull request whose required CI checks are green, after a maintainer's review.
- **Your branch**: create it from `main` in your fork. Suggested prefixes: `feat/`, `fix/`, `docs/`, `test/`, `ci/`, `refactor/`, `experiment/` (research and prototypes).
- **One topic per pull request.** A large change is easier to review as several pull requests, each building and passing tests on its own.
- **Releases** are cut by the maintainers from `main`: a release pull request updates `CHANGELOG.md` and `ROADMAP.md`, then the tag is created on the merged commit.

```bash
# Fork go-webgpu/goffi on GitHub, then:
git clone https://github.com/<you>/goffi.git
cd goffi
git remote add upstream https://github.com/go-webgpu/goffi.git

git fetch upstream
git switch -c fix/my-fix upstream/main

# Work, test, commit...
git commit -m "fix(ffi): describe the fix"

# Keep the branch current by rebasing, not merging main into it
git fetch upstream
git rebase upstream/main

git push origin fix/my-fix
# Open a pull request against go-webgpu/goffi:main
```

For a bug fix, the best pull request first adds a test that fails on `main` and then the fix that makes it pass.

## Commit Message Guidelines

Follow [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

- **feat**: New feature (e.g., new platform support, API improvements)
- **fix**: Bug fix (e.g., assembly errors, race conditions)
- **docs**: Documentation changes
- **style**: Code style changes (formatting, etc.)
- **refactor**: Code refactoring (e.g., assembly optimization)
- **test**: Adding or updating tests
- **ci**: CI configuration
- **chore**: Maintenance tasks (build, dependencies, etc.)
- **perf**: Performance improvements (critical for FFI!)

### Examples

```bash
feat: add ARM64 Linux support with AAPCS64 ABI
fix: correct Win64 shadow space handling in assembly
docs: update PERFORMANCE.md with ARM64 benchmarks
refactor: optimize System V register loading sequence
test: add race detector tests for concurrent FFI calls
ci: run the test suite on FreeBSD in a VM
perf: reduce FFI overhead with assembly tweaks
```

## License Headers

goffi is MIT licensed. Every new source file (Go, assembly, scripts, tests) starts with an SPDX header:

```go
// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors
```

Use the Apache-2.0 header only for code actually copied or adapted from [purego](https://github.com/ebitengine/purego), and keep its `The Ebitengine Authors` copyright line; code derived from the Go runtime keeps the Go Authors' BSD header. Do not copy the header of a neighbouring file — the header follows where the code comes from. CI checks this (`scripts/check-spdx.sh`).

## Code Quality Standards

### Before Committing

1. **Format code**:
   ```bash
   gofmt -l $(git ls-files '*.go')   # must print nothing
   go fmt ./...
   ```

2. **Run linter** — on your host and on the other targets (goffi is pure Go, so `GOOS`/`GOARCH` select exactly the files each target compiles):
   ```bash
   golangci-lint run --config=.golangci.yml ./...

   for t in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
     GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 golangci-lint run --allow-serial-runners ./...
   done
   for a in amd64 arm64; do
     GOOS=freebsd GOARCH=$a CGO_ENABLED=0 \
       GOFLAGS=-gcflags=github.com/go-webgpu/goffi/internal/fakecgo=-std \
       golangci-lint run --allow-serial-runners ./...
   done
   ```
   Never lint another target with `--build-tags linux`: a GOOS-named tag adds `*_linux.go` files to the host's files instead of switching the target.

3. **Run tests** (C-backed end-to-end tests need a C compiler for the target, otherwise they skip):
   ```bash
   CGO_ENABLED=0 go test ./...
   CGO_ENABLED=1 go test ./...
   ```

4. **Run tests with race detector** (needs cgo and a C compiler — Linux, macOS, or WSL on Windows):
   ```bash
   CGO_ENABLED=1 go test -race ./...
   ```

5. **Run benchmarks** (FFI performance critical!):
   ```bash
   go test -run='^$' -bench=BenchmarkGoffi -benchmem ./ffi
   ```

6. **Pre-release check** (comprehensive):
   ```bash
   bash scripts/pre-release-check.sh
   ```

### Pull Request Requirements

- [ ] Code is formatted (`gofmt` prints nothing)
- [ ] Linter passes on every target (see above)
- [ ] All tests pass, including with the race detector where available
- [ ] Bug fixes come with a regression test that fails without the fix
- [ ] Benchmarks don't regress (FFI overhead < 200ns)
- [ ] New code has tests (CI gate: 70% coverage)
- [ ] Platform-specific code tested on target OS (CI runs native Linux, Windows and macOS runners on amd64 and arm64, plus FreeBSD VMs)
- [ ] Assembly changes validated on real hardware
- [ ] New files carry the MIT SPDX header (see [License Headers](#license-headers))
- [ ] Documentation updated (if applicable)
- [ ] Commit messages follow conventions
- [ ] No sensitive data (credentials, tokens, etc.)

## Development Setup

### Prerequisites

- **Go 1.25 or later** (the `go` line in `go.mod`)
- **golangci-lint v2** (code quality)
- **A C compiler** (gcc or clang) — optional; needed for the race detector, `CGO_ENABLED=1` builds and the C-backed end-to-end tests
- **Platform access**:
  - Linux, macOS or Windows; WSL2 works for Linux testing on Windows
  - CI covers the other targets

### Install Dependencies

```bash
# Install golangci-lint v2
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

# Verify installation
golangci-lint --version
go version  # Should be 1.25+
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detector (CRITICAL for FFI!)
CGO_ENABLED=1 go test -race ./...

# Fail instead of skipping when the C test library cannot be built (what CI does)
GOFFI_REQUIRE_TESTLIB=1 go test ./ffi

# Run benchmarks
go test -run='^$' -bench=. -benchmem ./ffi

# Cross-compile check for another target (tests run only on that target)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build ./...
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go test -c -o /dev/null ./ffi
```

### Running Linter

```bash
# Run linter
golangci-lint run --config=.golangci.yml ./...

# Run with verbose output
golangci-lint run --config=.golangci.yml --verbose ./...
```

## Project Structure

```
goffi/
├── .github/
│   ├── CODEOWNERS           # Code ownership
│   └── workflows/
│       └── ci.yml           # GitHub Actions
├── .golangci.yml            # Linter configuration
├── .codecov.yml             # Codecov configuration
├── cmd/
│   └── variadic-test/       # Variadic ABI check binary
├── docs/                    # ARCHITECTURE, PERFORMANCE, ANDROID, ADR-001
├── examples/
│   ├── simple/              # Minimal usage
│   └── struct/              # Struct pass/return
├── ffi/                     # Public API: LoadLibrary, PrepareCallInterface, CallFunction, NewCallback
│   ├── callback_*.s         # Callback trampolines (Unix)
│   └── testdata/structtest.c # C library for end-to-end tests
├── types/                   # Type descriptors, CallInterface
├── internal/
│   ├── arch/                # ABI classification and argument packing (amd64, arm64)
│   ├── syscall/             # Assembly call path + errno capture (System V amd64, arm64)
│   ├── dl/                  # dlopen/dlsym/dlerror bindings (Unix)
│   ├── fakecgo/             # cgo runtime for CGO_ENABLED=0 (Unix)
│   └── runtime/             # Legacy Linux asmcgocall hook (not imported by any package)
├── scripts/                 # pre-release-check, check-spdx, ELF/Android/FreeBSD checks
├── CHANGELOG.md             # Version history
├── ROADMAP.md               # Development roadmap
├── CONTRIBUTING.md          # This file
├── CODE_OF_CONDUCT.md       # Community standards
├── SECURITY.md              # Security policy
└── README.md                # Main documentation
```

## Adding New Features

1. Check if issue exists, if not create one
2. Discuss approach in the issue (especially for assembly/platform changes!)
3. Create a branch from `main`
4. Implement feature with tests
5. **Test on target platform** (critical for FFI!)
6. Run benchmarks to ensure no performance regression
7. Update documentation (README.md, docs/, CHANGELOG.md `[Unreleased]`)
8. Run quality checks (`scripts/pre-release-check.sh`)
9. Create a pull request to `main`
10. Wait for code review
11. Address feedback by pushing commits to the same branch
12. A maintainer merges when CI is green and the review is approved

## Platform-Specific Development

### Adding New Platform Support

1. Research platform ABI (System V, Win64, AAPCS64, etc.) — the ABI specification and libffi are the references
2. Create an `experiment/platform-name` branch
3. Implement the call path in `internal/syscall/syscall_<os>_<arch>.s` (or reuse an existing one) and the argument packing in `internal/arch/<arch>/`
4. Register the implementation in `internal/arch/` and add build constraints
5. Write platform-specific tests, including C-backed end-to-end tests
6. Run benchmarks on real hardware
7. Document limitations and ABI details
8. Update CHANGELOG.md with platform support

### Assembly Guidelines

- **Comment every instruction** - Assembly is hard to review!
- **Reference ABI specification** - Add links in comments
- **Test on real hardware** - VMs may hide bugs
- **Benchmark before/after** - Performance is critical
- **Handle edge cases** - Variadic, floats, large structs
- **Follow platform conventions** - Red zones, shadow space, etc.

Example assembly header:
```asm
// System V AMD64 ABI Implementation
// Specification: https://gitlab.com/x86-psABIs/x86-64-ABI
// Registers: RDI, RSI, RDX, RCX, R8, R9 (GP), XMM0-7 (FP)
// Red zone: 128 bytes below RSP
```

## Code Style Guidelines

### General Principles

- Follow Go conventions and idioms
- Write self-documenting code
- Add comments for complex logic (especially assembly!)
- Keep functions small and focused
- Use meaningful variable names
- **Performance matters** - This is a low-level FFI library

### Naming Conventions

- **Public types/functions**: `PascalCase` (e.g., `CallFunction`, `TypeDescriptor`)
- **Private types/functions**: `camelCase` (e.g., `loadLibrary`, `prepareArgs`)
- **Constants**: `PascalCase` with context (e.g., `DefaultCall`, `WindowsCallingConvention`)
- **Test functions**: `Test*` (e.g., `TestCallFunction`)
- **Benchmark functions**: `Benchmark*` (e.g., `BenchmarkGoffiOverhead`)

### Error Handling

- Use typed errors (`LibraryError`, `InvalidCallInterfaceError`, etc.)
- Always check and handle errors
- Add context with error wrapping: `fmt.Errorf("context: %w", err)`
- Never ignore errors
- Validate inputs (type safety critical for FFI!)

### Testing

- Use `testing` package (standard library)
- Test both success and error cases
- Use table-driven tests when appropriate
- **Always test with race detector** (`-race`) where available
- Hold Go pointers passed to C as `unsafe.Pointer`, never `uintptr` (a stack copy leaves a `uintptr` stale); use `runtime.Pinner` for memory C or a callback frame reads later
- Tests must build with the oldest supported Go (the `go` line in `go.mod`)
- Write benchmarks for performance-critical code
- Test on all supported platforms

### Performance Considerations

- **FFI overhead matters** - Every nanosecond counts
- Profile before optimizing: `go test -cpuprofile=cpu.prof`
- Benchmark before/after changes: `benchstat before.txt after.txt`
- Target: < 200ns FFI overhead (current: 88-114ns)
- Avoid allocations in hot paths
- Reuse `CallInterface` objects and `avalue` slices

## Getting Help

- Check existing issues and discussions
- Read documentation in `docs/`
- Review `CHANGELOG.md` for known limitations
- Review [ROADMAP.md](ROADMAP.md) and the [milestones](https://github.com/go-webgpu/goffi/milestones) for planned work
- Ask questions in GitHub Issues: https://github.com/go-webgpu/goffi/issues

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to goffi!** 🚀

*Building the future of Zero-CGO FFI for Go*
