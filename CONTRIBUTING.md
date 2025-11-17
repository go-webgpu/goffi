# Contributing to goffi - Zero-CGO FFI for Go

Thank you for considering contributing to goffi! This document outlines the development workflow and guidelines.

## Git Workflow (Git-Flow)

This project uses Git-Flow branching model for development.

### Branch Structure

```
main                 # Production-ready code (tagged releases)
  └─ develop         # Integration branch for next release
       ├─ feature/*  # New features
       ├─ experiment/* # FFI research and prototypes
       ├─ bugfix/*   # Bug fixes
       └─ hotfix/*   # Critical fixes from main
```

### Branch Purposes

- **main**: Production-ready code. Only releases are merged here.
- **develop**: Active development branch. All features merge here first.
- **feature/\***: New features. Branch from `develop`, merge back to `develop`.
- **experiment/\***: FFI research, platform support, assembly prototypes. Branch from `develop`.
- **bugfix/\***: Bug fixes. Branch from `develop`, merge back to `develop`.
- **hotfix/\***: Critical production fixes. Branch from `main`, merge to both `main` and `develop`.

### Workflow Commands

#### Starting a New Feature

```bash
# Create feature branch from develop
git checkout develop
git pull origin develop
git checkout -b feature/my-new-feature

# Work on your feature...
git add .
git commit -m "feat: add my new feature"

# When done, merge back to develop
git checkout develop
git merge --no-ff feature/my-new-feature
git branch -d feature/my-new-feature
git push origin develop
```

#### Experimenting with FFI (Assembly, Platform Support)

```bash
# Create experiment branch from develop
git checkout develop
git pull origin develop
git checkout -b experiment/arm64-support

# Prototype assembly, test platforms...
git add .
git commit -m "experiment: ARM64 AAPCS64 ABI prototype"

# Push to share progress
git push origin experiment/arm64-support

# When stable, merge to develop
git checkout develop
git merge --no-ff experiment/arm64-support
```

#### Fixing a Bug

```bash
# Create bugfix branch from develop
git checkout develop
git pull origin develop
git checkout -b bugfix/fix-issue-123

# Fix the bug...
git add .
git commit -m "fix: resolve issue #123"

# Merge back to develop
git checkout develop
git merge --no-ff bugfix/fix-issue-123
git branch -d bugfix/fix-issue-123
git push origin develop
```

#### Creating a Release

```bash
# Create release branch from develop
git checkout develop
git pull origin develop
git checkout -b release/v0.2.0

# Update version numbers, CHANGELOG, etc.
git add .
git commit -m "chore: prepare release v0.2.0"

# Merge to main and tag
git checkout main
git merge --no-ff release/v0.2.0
git tag -a v0.2.0 -m "Release v0.2.0"

# Merge back to develop
git checkout develop
git merge --no-ff release/v0.2.0

# Delete release branch
git branch -d release/v0.2.0

# Push everything
git push origin main develop --tags
```

#### Hotfix (Critical Production Bug)

```bash
# Create hotfix branch from main
git checkout main
git pull origin main
git checkout -b hotfix/critical-bug

# Fix the bug...
git add .
git commit -m "fix: critical production bug"

# Merge to main and tag
git checkout main
git merge --no-ff hotfix/critical-bug
git tag -a v0.1.1 -m "Hotfix v0.1.1"

# Merge to develop
git checkout develop
git merge --no-ff hotfix/critical-bug

# Delete hotfix branch
git branch -d hotfix/critical-bug

# Push everything
git push origin main develop --tags
```

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
- **chore**: Maintenance tasks (build, dependencies, etc.)
- **perf**: Performance improvements (critical for FFI!)
- **experiment**: Research and prototypes (platform support, new ABIs)

### Examples

```bash
feat: add ARM64 Linux support with AAPCS64 ABI
fix: correct Win64 shadow space handling in assembly
docs: update PERFORMANCE.md with ARM64 benchmarks
refactor: optimize System V register loading sequence
test: add race detector tests for concurrent FFI calls
chore: update golangci-lint configuration
perf: reduce FFI overhead to 80ns with assembly tweaks
experiment: prototype macOS Apple Silicon support
```

## Code Quality Standards

### Before Committing

1. **Format code**:
   ```bash
   go fmt ./...
   ```

2. **Run linter**:
   ```bash
   golangci-lint run --config=.golangci.yml ./...
   ```

3. **Run tests with race detector**:
   ```bash
   go test -race ./...
   ```

4. **Run benchmarks** (FFI performance critical!):
   ```bash
   go test -bench=BenchmarkGoffi -benchmem ./ffi
   ```

5. **Pre-release check** (comprehensive):
   ```bash
   bash scripts/pre-release-check.sh
   ```

### Pull Request Requirements

- [ ] Code is formatted (`go fmt ./...`)
- [ ] Linter passes (`golangci-lint run`)
- [ ] All tests pass with race detector (`go test -race ./...`)
- [ ] Benchmarks don't regress (FFI overhead < 200ns)
- [ ] New code has tests (minimum 70% coverage, current: 89.1%)
- [ ] Platform-specific code tested on target OS
- [ ] Assembly changes validated on real hardware
- [ ] Documentation updated (if applicable)
- [ ] Commit messages follow conventions
- [ ] No sensitive data (credentials, tokens, etc.)

## Development Setup

### Prerequisites

- **Go 1.25 or later** (required for latest runtime.cgocall features)
- **golangci-lint** (code quality)
- **GCC or Clang** (for race detector, optional but recommended)
- **Platform access**:
  - Linux AMD64 (primary development)
  - Windows AMD64 (cross-platform testing)
  - WSL2 with Go (alternative for Windows developers)

### Install Dependencies

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

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
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem ./ffi

# Run platform-specific tests
GOOS=linux GOARCH=amd64 go test ./...
GOOS=windows GOARCH=amd64 go test ./...
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
│   └── workflows/           # CI/CD
│       └── ci.yml          # GitHub Actions
├── .golangci.yml            # Linter configuration
├── .codecov.yml             # Codecov configuration
├── scripts/
│   └── pre-release-check.sh # Pre-release validation
├── docs/
│   └── PERFORMANCE.md       # Performance guide & benchmarks
├── examples/
│   └── simple/             # Usage examples
├── experiments/             # Research & prototypes (not production)
├── ffi/                     # Core FFI package
│   ├── ffi.go              # Public API
│   ├── ffi_test.go         # Tests
│   └── benchmark_test.go   # Performance benchmarks
├── types/                   # Type system
│   ├── types.go            # Type descriptors
│   └── types_test.go       # Type tests
├── internal/
│   ├── arch/               # Architecture-specific code
│   ├── syscall/            # Low-level syscalls
│   │   ├── call_unix.s     # System V AMD64 assembly
│   │   └── call_windows.s  # Win64 assembly
│   ├── runtime/            # Go runtime integration
│   ├── fakecgo/            # runtime.cgocall implementation
│   └── dl/                 # Dynamic library loading
├── reference/              # External reference code (gitignored)
│   ├── libffi/            # libffi reference
│   └── purego/            # purego comparison
├── CHANGELOG.md            # Version history
├── API_TODO.md             # API improvement backlog
├── CONTRIBUTING.md         # This file
├── CODE_OF_CONDUCT.md      # Community standards
└── README.md               # Main documentation
```

## Adding New Features

1. Check if issue exists, if not create one
2. Discuss approach in the issue (especially for assembly/platform changes!)
3. Create feature or experiment branch from `develop`
4. Implement feature with tests
5. **Test on target platform** (critical for FFI!)
6. Run benchmarks to ensure no performance regression
7. Update documentation (README.md, PERFORMANCE.md, etc.)
8. Run quality checks (`scripts/pre-release-check.sh`)
9. Create pull request to `develop`
10. Wait for code review
11. Address feedback
12. Merge when approved

## Platform-Specific Development

### Adding New Platform Support

1. Research platform ABI (System V, Win64, AAPCS64, etc.)
2. Create `experiment/platform-name` branch
3. Implement assembly in `internal/syscall/call_platform.s`
4. Add platform detection in `internal/arch/`
5. Write platform-specific tests
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
// Specification: https://refspecs.linuxbase.org/elf/x86_64-abi-0.99.pdf
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
- **Always test with race detector** (`-race`)
- Write benchmarks for performance-critical code
- Test on all supported platforms

### Performance Considerations

- **FFI overhead matters** - Every nanosecond counts
- Profile before optimizing: `go test -cpuprofile=cpu.prof`
- Benchmark before/after changes: `benchstat before.txt after.txt`
- Target: < 200ns FFI overhead (current: 88-114ns)
- Avoid allocations in hot paths
- Reuse `CallInterface` objects

## Getting Help

- Check existing issues and discussions
- Read documentation in `docs/`
- Review `CHANGELOG.md` for known limitations
- Review `API_TODO.md` for planned improvements
- Ask questions in GitHub Issues: https://github.com/go-webgpu/goffi/issues

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to goffi!** 🚀

*Building the future of Zero-CGO FFI for Go*
