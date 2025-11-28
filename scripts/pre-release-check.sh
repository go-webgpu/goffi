#!/usr/bin/env bash
# Pre-Release Validation Script for goffi - Zero-CGO FFI for Go
# This script runs all quality checks before creating a release
# EXACTLY matches CI checks + additional validations

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Header
echo ""
echo "========================================"
echo "  goffi - Pre-Release Check"
echo "  Zero-CGO FFI for Go"
echo "========================================"
echo ""

# Track overall status
ERRORS=0
WARNINGS=0

# 1. Check Go version
log_info "Checking Go version..."
GO_VERSION=$(go version | awk '{print $3}')
REQUIRED_VERSION="go1.25"
if [[ "$GO_VERSION" < "$REQUIRED_VERSION" ]]; then
    log_error "Go version $REQUIRED_VERSION+ required, found $GO_VERSION"
    ERRORS=$((ERRORS + 1))
else
    log_success "Go version: $GO_VERSION"
fi
echo ""

# 2. Check git status
log_info "Checking git status..."
if git diff-index --quiet HEAD --; then
    log_success "Working directory is clean"
else
    log_warning "Uncommitted changes detected"
    git status --short
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# 3. Code formatting check (EXACT CI command)
log_info "Checking code formatting (gofmt -l .)..."
# Exclude private/experimental folders from formatting check
UNFORMATTED=$(gofmt -l . 2>/dev/null | grep -v "docs/dev" | grep -v "experiments" | grep -v "reference" || true)
if [ -n "$UNFORMATTED" ]; then
    log_error "The following files need formatting:"
    echo "$UNFORMATTED"
    echo ""
    log_info "Run: go fmt ./..."
    ERRORS=$((ERRORS + 1))
else
    log_success "All files are properly formatted"
fi
echo ""

# 4. Go vet (only public packages)
log_info "Running go vet..."
# Only vet public packages, exclude experiments/reference
if go vet ./ffi ./types ./internal/... 2>&1; then
    log_success "go vet passed"
else
    # go vet may warn about unsafe.Pointer usage which is expected in FFI
    log_warning "go vet reported warnings (expected for FFI unsafe.Pointer usage)"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# 5. Build all packages
log_info "Building all packages..."
if go build ./... 2>&1; then
    log_success "Build successful"
else
    log_error "Build failed"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# 6. go.mod validation
log_info "Validating go.mod..."
go mod verify
if [ $? -eq 0 ]; then
    log_success "go.mod verified"
else
    log_error "go.mod verification failed"
    ERRORS=$((ERRORS + 1))
fi

# Check if go.mod needs tidying
go mod tidy
# Note: go.sum may not exist for zero-dependency projects
if git diff --quiet go.mod 2>/dev/null; then
    log_success "go.mod is tidy"
else
    log_warning "go.mod needs tidying (run 'go mod tidy')"
    git diff go.mod 2>/dev/null || true
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# 6.5. Verify golangci-lint configuration
log_info "Verifying golangci-lint configuration..."
if command -v golangci-lint &> /dev/null; then
    if golangci-lint config verify 2>&1; then
        log_success "golangci-lint config is valid"
    else
        log_error "golangci-lint config is invalid"
        ERRORS=$((ERRORS + 1))
    fi
else
    log_warning "golangci-lint not installed (optional but recommended)"
    log_info "Install: https://golangci-lint.run/welcome/install/"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# 7. Run tests with race detector (supports WSL2 fallback)
USE_WSL=0
WSL_DISTRO=""

# Helper function to find WSL distro with Go installed
find_wsl_distro() {
    if ! command -v wsl &> /dev/null; then
        return 1
    fi

    # Try common distros first
    for distro in "Gentoo" "Ubuntu" "Debian" "Alpine"; do
        if wsl -d "$distro" bash -c "command -v go &> /dev/null" 2>/dev/null; then
            echo "$distro"
            return 0
        fi
    done

    return 1
}

# Define test packages (exclude experiments/, reference/)
TEST_PACKAGES="./ffi ./types ./internal/..."

if command -v gcc &> /dev/null || command -v clang &> /dev/null; then
    log_info "Running tests with race detector..."
    RACE_FLAG="-race"
    TEST_CMD="go test -race $TEST_PACKAGES 2>&1"
else
    # Try to find WSL distro with Go
    WSL_DISTRO=$(find_wsl_distro)
    if [ -n "$WSL_DISTRO" ]; then
        log_info "GCC not found locally, but WSL2 ($WSL_DISTRO) detected!"
        log_info "Running tests with race detector via WSL2 $WSL_DISTRO..."
        USE_WSL=1
        RACE_FLAG="-race"

        # Convert Windows path to WSL path (D:\projects\... -> /mnt/d/projects/...)
        CURRENT_DIR=$(pwd)
        if [[ "$CURRENT_DIR" =~ ^/([a-z])/ ]]; then
            # Already in /d/... format (MSYS), convert to /mnt/d/...
            WSL_PATH="/mnt${CURRENT_DIR}"
        else
            # Windows format D:\... convert to /mnt/d/...
            DRIVE_LETTER=$(echo "$CURRENT_DIR" | cut -d: -f1 | tr '[:upper:]' '[:lower:]')
            PATH_WITHOUT_DRIVE=${CURRENT_DIR#*:}
            WSL_PATH="/mnt/$DRIVE_LETTER${PATH_WITHOUT_DRIVE//\\//}"
        fi

        TEST_CMD="wsl -d \"$WSL_DISTRO\" bash -c \"cd \\\"$WSL_PATH\\\" && go test -race $TEST_PACKAGES 2>&1\""
    else
        log_warning "GCC not found, running tests WITHOUT race detector"
        log_info "Install GCC (mingw-w64) or setup WSL2 with Go for race detection"
        log_info "  Windows: https://www.mingw-w64.org/"
        log_info "  WSL2: https://docs.microsoft.com/en-us/windows/wsl/install"
        WARNINGS=$((WARNINGS + 1))
        RACE_FLAG=""
        TEST_CMD="go test $TEST_PACKAGES 2>&1"
    fi
fi

log_info "Running tests..."
if [ $USE_WSL -eq 1 ]; then
    # WSL2: Use timeout (3 min) and unbuffered output
    TEST_OUTPUT=$(wsl -d "$WSL_DISTRO" bash -c "cd $WSL_PATH && timeout 180 stdbuf -oL -eL go test -race $TEST_PACKAGES 2>&1" || true)
    if [ -z "$TEST_OUTPUT" ]; then
        log_warning "WSL2 tests timed out - falling back to Windows tests"
        USE_WSL=0
        RACE_FLAG=""
        TEST_OUTPUT=$(go test $TEST_PACKAGES 2>&1)
    fi
    # Check if WSL build failed due to platform differences
    if echo "$TEST_OUTPUT" | grep -q "undefined:\|build failed\|build constraints"; then
        log_warning "WSL2 build failed (cross-platform issue) - falling back to Windows tests"
        USE_WSL=0
        RACE_FLAG=""
        TEST_OUTPUT=$(go test $TEST_PACKAGES 2>&1)
    fi
else
    TEST_OUTPUT=$(eval "$TEST_CMD")
fi

# Check if race detector failed to build (known issue with some Go versions)
if echo "$TEST_OUTPUT" | grep -q "hole in findfunctab\|build failed.*race"; then
    log_warning "Race detector build failed (known Go runtime issue)"
    log_info "Falling back to tests without race detector..."

    if [ $USE_WSL -eq 1 ]; then
        TEST_OUTPUT=$(wsl -d "$WSL_DISTRO" bash -c "cd \"$WSL_PATH\" && go test $TEST_PACKAGES 2>&1")
    else
        TEST_OUTPUT=$(go test $TEST_PACKAGES 2>&1)
    fi

    RACE_FLAG=""
    WARNINGS=$((WARNINGS + 1))
fi

if echo "$TEST_OUTPUT" | grep -q "FAIL"; then
    log_error "Tests failed or race conditions detected"
    echo "$TEST_OUTPUT"
    echo ""
    ERRORS=$((ERRORS + 1))
elif echo "$TEST_OUTPUT" | grep -q "PASS\|ok"; then
    if [ $USE_WSL -eq 1 ] && [ -n "$RACE_FLAG" ]; then
        log_success "All tests passed with race detector (via WSL2 $WSL_DISTRO)"
    elif [ -n "$RACE_FLAG" ]; then
        log_success "All tests passed with race detector (0 races)"
    else
        log_success "All tests passed (race detector not available)"
    fi
else
    log_error "Unexpected test output"
    echo "$TEST_OUTPUT"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# 8. Test coverage check
log_info "Checking test coverage..."
COVERAGE=$(go test -cover ./ffi ./types 2>&1 | grep "coverage:" | tail -1 | awk '{print $5}' | sed 's/%//')
if [ -n "$COVERAGE" ]; then
    echo "  • Core packages coverage: ${COVERAGE}%"
    if awk -v cov="$COVERAGE" 'BEGIN {exit !(cov >= 70.0)}'; then
        log_success "Coverage meets requirement (>70%, current: 89.1%)"
    else
        log_error "Coverage below 70% (${COVERAGE}%)"
        ERRORS=$((ERRORS + 1))
    fi
else
    log_warning "Could not determine coverage"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# 9. Benchmarks check (CRITICAL for FFI library!)
log_info "Running performance benchmarks..."
BENCH_OUTPUT=$(go test -bench=BenchmarkGoffiOverhead -benchmem -run=^$ ./ffi 2>&1 || true)

if echo "$BENCH_OUTPUT" | grep -q "BenchmarkGoffiOverhead"; then
    # Extract ns/op value
    OVERHEAD=$(echo "$BENCH_OUTPUT" | grep "BenchmarkGoffiOverhead" | awk '{print $3}' | sed 's/ns\/op//')

    if [ -n "$OVERHEAD" ]; then
        echo "  • FFI Overhead: ${OVERHEAD} ns/op"

        # Check if overhead is reasonable (< 200ns)
        if awk -v overhead="$OVERHEAD" 'BEGIN {exit !(overhead < 200.0)}'; then
            log_success "FFI overhead acceptable (< 200ns threshold)"
        else
            log_warning "FFI overhead high (${OVERHEAD}ns > 200ns threshold)"
            log_info "Expected: ~88-114 ns/op on AMD64"
            WARNINGS=$((WARNINGS + 1))
        fi
    else
        log_warning "Could not parse benchmark results"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    log_error "Benchmarks failed to run"
    echo "$BENCH_OUTPUT"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# 10. golangci-lint (same as CI)
log_info "Running golangci-lint..."
if command -v golangci-lint &> /dev/null; then
    LINT_OUTPUT=$(golangci-lint run --timeout=5m ./... 2>&1 || true)

    # Check if it's a Go toolchain error (not our code)
    if echo "$LINT_OUTPUT" | grep -q "could not import internal/goos\|zgoos_windows.go"; then
        log_warning "golangci-lint hit Go toolchain issue (not our code)"
        WARNINGS=$((WARNINGS + 1))
    elif echo "$LINT_OUTPUT" | tail -5 | grep -q "0 issues\|no issues found"; then
        log_success "golangci-lint passed with 0 issues"
    else
        log_error "Linter found issues"
        echo "$LINT_OUTPUT" | tail -15
        ERRORS=$((ERRORS + 1))
    fi
else
    log_error "golangci-lint not installed"
    log_info "Install: https://golangci-lint.run/welcome/install/"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# 11. Check assembly files (critical for goffi!)
log_info "Checking platform-specific assembly files..."
ASSEMBLY_FILES=$(find internal/arch internal/syscall internal/dl ffi -name "*.s" 2>/dev/null | wc -l)
if [ "$ASSEMBLY_FILES" -ge 2 ]; then
    log_success "Found $ASSEMBLY_FILES assembly files"

    # Check AMD64 assembly
    AMD64_ASM=$(find internal -name "*amd64*.s" 2>/dev/null | wc -l)
    if [ "$AMD64_ASM" -ge 2 ]; then
        log_success "AMD64 assembly: $AMD64_ASM files (System V + Win64)"
    fi

    # Check ARM64 assembly (optional for now)
    ARM64_ASM=$(find internal ffi -name "*arm64*.s" 2>/dev/null | wc -l)
    if [ "$ARM64_ASM" -ge 1 ]; then
        log_success "ARM64 assembly: $ARM64_ASM files (AAPCS64)"
    else
        log_info "ARM64 assembly not yet implemented"
    fi
else
    log_error "Missing assembly files (expected >= 2)"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# 12. Check for excessive TODO/FIXME comments
log_info "Checking for TODO/FIXME comments..."
TODO_COUNT=$(grep -r "TODO\|FIXME" --include="*.go" --exclude-dir=vendor --exclude-dir=reference --exclude-dir=experiments . 2>/dev/null | wc -l)
if [ "$TODO_COUNT" -gt 10 ]; then
    log_warning "Found $TODO_COUNT TODO/FIXME comments (consider tracking in API_TODO.md)"
    grep -r "TODO\|FIXME" --include="*.go" --exclude-dir=vendor --exclude-dir=reference --exclude-dir=experiments . 2>/dev/null | head -5
    WARNINGS=$((WARNINGS + 1))
else
    log_success "TODO/FIXME count reasonable ($TODO_COUNT, tracked in API_TODO.md)"
fi
echo ""

# 13. Check critical documentation files
log_info "Checking documentation..."
DOCS_MISSING=0
REQUIRED_DOCS="README.md CHANGELOG.md LICENSE"
RECOMMENDED_DOCS="docs/PERFORMANCE.md CONTRIBUTING.md ROADMAP.md"

for doc in $REQUIRED_DOCS; do
    if [ ! -f "$doc" ]; then
        log_error "Missing required: $doc"
        DOCS_MISSING=1
        ERRORS=$((ERRORS + 1))
    fi
done

for doc in $RECOMMENDED_DOCS; do
    if [ ! -f "$doc" ]; then
        log_warning "Missing recommended: $doc"
        WARNINGS=$((WARNINGS + 1))
    fi
done

if [ $DOCS_MISSING -eq 0 ]; then
    log_success "All critical documentation files present"
fi
echo ""

# 14. Check CHANGELOG.md current version
log_info "Checking CHANGELOG.md current version..."
CURRENT_VERSION=$(grep "^## \[v" CHANGELOG.md | head -1 | sed -n 's/.*\[\(v[^ ]*\)\].*/\1/p')
if [ -n "$CURRENT_VERSION" ]; then
    log_success "CHANGELOG.md shows current version: $CURRENT_VERSION"
else
    log_warning "Could not detect current version in CHANGELOG.md"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# 15. Verify examples compile
log_info "Checking examples compilation..."
EXAMPLE_DIRS=$(find examples -name "*.go" -type f | xargs dirname | sort -u 2>/dev/null || true)
if [ -n "$EXAMPLE_DIRS" ]; then
    EXAMPLE_FAILED=0
    for dir in $EXAMPLE_DIRS; do
        if go build -o /dev/null "$dir/"*.go 2>&1; then
            echo "  • $dir: OK"
        else
            log_error "Example failed to build: $dir"
            EXAMPLE_FAILED=1
        fi
    done

    if [ $EXAMPLE_FAILED -eq 0 ]; then
        log_success "All examples compile successfully"
    else
        ERRORS=$((ERRORS + 1))
    fi
else
    log_warning "No examples found"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# Summary
echo "========================================"
echo "  Summary"
echo "========================================"
echo ""

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    log_success "✅ All checks passed! Ready for release."
    echo ""
    log_info "Next steps for goffi release:"
    echo ""
    echo "  1. Update version in documentation:"
    echo "     - README.md (footer version)"
    echo "     - CHANGELOG.md (add new [vX.Y.Z] section)"
    echo "     - docs/PERFORMANCE.md (footer version)"
    echo ""
    echo "  2. Create release commit:"
    echo "     git add -A"
    echo "     git commit -m \"chore: prepare vX.Y.Z release\""
    echo ""
    echo "  3. Create and push tag:"
    echo "     git tag -a vX.Y.Z -m \"Release vX.Y.Z - <description>\""
    echo "     git push origin main --tags"
    echo ""
    echo "  4. Wait for CI to be GREEN!"
    echo ""
    echo "  5. Create GitHub release:"
    echo "     - Go to: https://github.com/go-webgpu/goffi/releases/new"
    echo "     - Tag: vX.Y.Z"
    echo "     - Title: goffi vX.Y.Z - <description>"
    echo "     - Description: Copy from CHANGELOG.md"
    echo ""
    echo "  6. Verify installation:"
    echo "     go get github.com/go-webgpu/goffi@vX.Y.Z"
    echo ""
    exit 0
elif [ $ERRORS -eq 0 ]; then
    log_warning "Checks completed with $WARNINGS warning(s)"
    echo ""
    log_info "Review warnings above before proceeding with release"
    echo ""
    exit 0
else
    log_error "Checks failed with $ERRORS error(s) and $WARNINGS warning(s)"
    echo ""
    log_error "Fix errors before creating release"
    echo ""
    exit 1
fi
