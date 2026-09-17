#!/usr/bin/env bash
# check-callback-trampolines.sh — assert that every target whose NewCallback is
# built from goffi's own assembly links ffi.callbackTrampoline as code.
#
# ffi/callback.go and ffi/callback_arm64.go take the address of the trampoline
# table through a //go:linkname *variable*. If the matching .s file is excluded
# by a narrower build constraint than the .go file, that variable itself
# becomes the definition of ffi.callbackTrampoline: the package still compiles
# and links, `go tool nm` reports D (data) instead of T (text), and
# NewCallback hands C code a pointer into a data page. freebsd/amd64 was built
# that way while callback_amd64.s lacked `freebsd` in its constraint, and no
# compile step could notice.
#
# The check links a real consumer binary per target, because building ./...
# links nothing, and inspects the symbol type.
#
# Usage: scripts/check-callback-trampolines.sh

set -euo pipefail

MODULE="github.com/go-webgpu/goffi"
SYMBOL="${MODULE}/ffi.callbackTrampoline"
FAKECGO_STD="-gcflags=${MODULE}/internal/fakecgo=-std"

# goos/goarch|extra go build flags
TARGETS=(
	"linux/amd64|"
	"linux/arm64|"
	"darwin/amd64|"
	"darwin/arm64|"
	"freebsd/amd64|${FAKECGO_STD}"
	"freebsd/arm64|${FAKECGO_STD}"
)

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GO_VERSION="$(cd "${ROOT}" && go list -m -f '{{.GoVersion}}')"

PROBE="$(mktemp -d)"
trap 'rm -rf "${PROBE}"' EXIT

cat >"${PROBE}/go.mod" <<EOF
module goffitrampolineprobe

go ${GO_VERSION}

require ${MODULE} v0.0.0

replace ${MODULE} => ${ROOT}
EOF

cat >"${PROBE}/main.go" <<'EOF'
package main

import "github.com/go-webgpu/goffi/ffi"

func main() {
	// A live reference keeps the trampoline table in the binary. Without it,
	// dead-code elimination drops the symbol and the check has nothing to see.
	_ = ffi.NewCallback(func(a uintptr) uintptr { return a })
}
EOF

FAILED=0
for entry in "${TARGETS[@]}"; do
	IFS='|' read -r target flags <<<"${entry}"
	goos="${target%%/*}"
	goarch="${target##*/}"
	bin="${PROBE}/probe-${goos}-${goarch}"

	# shellcheck disable=SC2086 # flags is either empty or a single word
	if ! (cd "${PROBE}" && CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
		go build ${flags} -o "${bin}" .); then
		echo "❌ ${target}: consumer probe failed to build"
		FAILED=1
		continue
	fi

	kind="$(go tool nm "${bin}" | awk -v s="${SYMBOL}" '$NF == s { print $(NF-1) }')"
	if [ "${kind}" = "T" ]; then
		echo "✅ ${target}: ffi.callbackTrampoline is code (T)"
	else
		echo "❌ ${target}: ffi.callbackTrampoline has type '${kind:-missing}', want T"
		echo "   Check that the build constraint of the callback .s file covers this target."
		FAILED=1
	fi
done

if [ "${FAILED}" -ne 0 ]; then
	exit 1
fi
echo "✅ Callback trampolines link as code on all asm-trampoline targets"
