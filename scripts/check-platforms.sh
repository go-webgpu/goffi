#!/usr/bin/env bash
#
# check-platforms.sh — the single source of truth for goffi's platform matrix.
#
# Every target purego supports under CGO_ENABLED=0 appears in the table below
# with the tier goffi actually delivers on it. The script then proves the tier:
#
#   full     dlopen + CallFunction + NewCallback (hand-written asm trampolines)
#   nocb     dlopen + CallFunction; NewCallback fails explicitly
#   load     dlopen + NewCallback (via syscall); CallFunction returns
#            ErrUnsupportedArchitecture because no ABI backend exists for the arch
#   pending  not supported yet — the build is EXPECTED to fail
#
# A "pending" target that starts building is a failure of this script too: it
# means docs/PLATFORMS.md and README.md are stale and someone forgot to promote
# the target. That keeps the published table honest without manual auditing.
#
# CGO_ENABLED=1-only targets from purego's matrix (iOS, Android amd64/386/arm)
# are deliberately absent: goffi's contract is a zero-CGO build.
#
# Usage: scripts/check-platforms.sh [--verbose]

set -uo pipefail

MODULE="github.com/go-webgpu/goffi"
FAKECGO_STD="-gcflags=${MODULE}/internal/fakecgo=-std"
VERBOSE="${1:-}"

cd "$(dirname "$0")/.."
ROOT="$(pwd)"

# ---------------------------------------------------------------------------
# The matrix. Format: goos/goarch|tier|extra go build flags
#
# NetBSD and FreeBSD need the fakecgo=-std gcflag because their fakecgo shims
# use //go:cgo_export_dynamic to publish environ/__progname (and __ps_strings on
# NetBSD) into the dynamic symbol table; rtld resolves libc's undefined
# references against them at startup.
# ---------------------------------------------------------------------------
TARGETS=(
  # --- Tier 1: full FFI, exercised by the runtime test matrix -------------
  "linux/amd64|full|"
  "linux/arm64|full|"
  "darwin/amd64|full|"
  "darwin/arm64|full|"
  "windows/amd64|full|"
  "windows/arm64|full|"

  # --- Tier 2: full FFI, cross-compile verified --------------------------
  "freebsd/amd64|full|${FAKECGO_STD}"
  "freebsd/arm64|full|${FAKECGO_STD}"
  "netbsd/amd64|full|${FAKECGO_STD}"
  "netbsd/arm64|full|${FAKECGO_STD}"

  # --- Tier 2: reduced capability ----------------------------------------
  # Android arm64 keeps its own deeper gate in check-android-arm64.sh (NDK
  # probes, ELF checks). Here we only prove it still compiles.
  "android/arm64|nocb|"
  # windows/386 mirrors purego's own note 6: no ABI backend for 386, so
  # CallFunction is unavailable, but LoadLibrary/GetSymbol/NewCallback work.
  "windows/386|load|"

  # --- Pending: purego supports these under CGO_ENABLED=0, goffi does not -
  # Each needs a per-arch call trampoline, errno capture, fakecgo bring-up and
  # dl stubs. See ROADMAP.md "Architecture expansion".
  "linux/386|pending|"
  "linux/arm|pending|"
  "linux/loong64|pending|"
  "linux/ppc64le|pending|"
  "linux/riscv64|pending|"
  "linux/s390x|pending|"
)

# Targets whose C->Go callback path is built from goffi's own assembly
# trampolines rather than syscall.NewCallback. On these the linker must resolve
# ffi.callbackTrampoline to a TEXT symbol.
#
# This is a regression gate, not a formality. When callback_amd64.s carried a
# narrower build constraint than callback.go, freebsd/amd64 silently linked
# ffi.callbackTrampoline to the 1-byte //go:linkname *variable* instead of the
# trampoline table, so trampolineBaseAddr pointed at data. Everything compiled;
# every callback would have jumped into a data page at runtime.
needs_asm_trampolines() {
  case "$1" in
    linux/amd64|linux/arm64|darwin/amd64|darwin/arm64) return 0 ;;
    freebsd/amd64|freebsd/arm64|netbsd/amd64|netbsd/arm64) return 0 ;;
    *) return 1 ;;
  esac
}

# ---------------------------------------------------------------------------
# A throwaway consumer module. Building ./... does not link anything, and the
# symbol table is the whole point of the trampoline check, so link a real
# binary that imports ffi the way a downstream project does.
# ---------------------------------------------------------------------------
PROBE_DIR="$(mktemp -d)"
trap 'rm -rf "${PROBE_DIR}"' EXIT

cat > "${PROBE_DIR}/main.go" <<'PROBE'
package main

import (
	"fmt"

	"github.com/go-webgpu/goffi/ffi"
)

func main() {
	_, err := ffi.LoadLibrary("definitely-not-a-real-library")
	fmt.Println(err != nil)

	// Reference the callback entry point so the linker keeps the trampoline
	// table. Without a live reference, dead-code elimination drops it and the
	// symbol check below would pass vacuously on a broken build.
	fmt.Println(ffi.NewCallback(func(a uintptr) uintptr { return a }))
}
PROBE

GO_VERSION="$(go list -m -f '{{.GoVersion}}')"

cat > "${PROBE_DIR}/go.mod" <<PROBE
module goffiplatformprobe

go ${GO_VERSION}

require ${MODULE} v0.0.0

replace ${MODULE} => ${ROOT}
PROBE

FAILED=0
declare -a SUMMARY

log() { [ -n "${VERBOSE}" ] && echo "      $*"; return 0; }

for entry in "${TARGETS[@]}"; do
  IFS='|' read -r target tier flags <<< "${entry}"
  goos="${target%%/*}"
  goarch="${target##*/}"

  # shellcheck disable=SC2086 # flags is intentionally word-split (may be empty)
  out=$(GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 go build ${flags} ./... 2>&1)
  rc=$?

  if [ "${tier}" = "pending" ]; then
    if [ ${rc} -eq 0 ]; then
      echo "  ❌ ${target} builds but is listed as pending"
      echo "     Promote it in scripts/check-platforms.sh, docs/PLATFORMS.md and README.md."
      FAILED=1
      SUMMARY+=("STALE   ${target}")
    else
      echo "  ⏳ ${target} pending (expected)"
      log "${out}" | head -3
      SUMMARY+=("pending ${target}")
    fi
    continue
  fi

  if [ ${rc} -ne 0 ]; then
    echo "  ❌ ${target} FAILED to build"
    echo "${out}" | head -20
    FAILED=1
    SUMMARY+=("FAIL    ${target}")
    continue
  fi

  # Link a real consumer binary and inspect what the linker produced.
  bin="${PROBE_DIR}/probe-${goos}-${goarch}"
  # shellcheck disable=SC2086
  link_out=$(cd "${PROBE_DIR}" && GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 GOFLAGS=-mod=mod \
    go build ${flags} -o "${bin}" . 2>&1)
  if [ $? -ne 0 ]; then
    echo "  ❌ ${target} compiles but a consumer binary does not link"
    echo "${link_out}" | head -20
    FAILED=1
    SUMMARY+=("LINK    ${target}")
    continue
  fi

  if needs_asm_trampolines "${target}"; then
    sym=$(go tool nm "${bin}" 2>/dev/null | awk '$3 == "github.com/go-webgpu/goffi/ffi.callbackTrampoline" {print $2}')
    if [ "${sym}" != "T" ]; then
      echo "  ❌ ${target} ffi.callbackTrampoline is '${sym:-missing}', expected 'T'"
      echo "     The callback assembly is excluded by its build constraint on this"
      echo "     target, so trampolineBaseAddr points at data instead of code."
      FAILED=1
      SUMMARY+=("TRAMP   ${target}")
      continue
    fi
    log "ffi.callbackTrampoline = T"
  fi

  echo "  ✅ ${target} (${tier})"
  SUMMARY+=("ok      ${target} [${tier}]")
done

echo ""
echo "Examples:"
for example_dir in examples/*/; do
  [ -f "${example_dir}go.mod" ] || continue
  name=$(basename "${example_dir}")
  if (cd "${example_dir}" && CGO_ENABLED=0 go build -o /dev/null . >/dev/null 2>&1); then
    echo "  ✅ ${name}"
  else
    echo "  ❌ ${name} FAILED"
    FAILED=1
  fi
done

echo ""
if [ ${FAILED} -eq 1 ]; then
  echo "❌ Platform matrix check FAILED"
  printf '%s\n' "${SUMMARY[@]}"
  exit 1
fi

echo "✅ Platform matrix check passed"
printf '   %s\n' "${SUMMARY[@]}"
