#!/bin/sh
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors
#
# run-linktest.sh — check that goffi links and works next to other code that
# calls C (#72), in every link mode the Go toolchain offers.
#
# testdata/linktest is a separate module (it depends on ebitengine/purego,
# goffi does not). The program calls getpid through goffi, purego and an
# ordinary cgo package, and checks goffi's errno capture.
#
#   CGO_ENABLED=1, default link mode   the cgo package and purego make cmd/link
#                                      use the external linker
#   CGO_ENABLED=1, -linkmode=external  explicit external linking
#   CGO_ENABLED=1, -buildmode=c-shared and c-archive (link only; both modes
#                                      always link externally)
#   CGO_ENABLED=0, -tags nofakecgo     the documented workaround for goffi +
#                                      purego without cgo (#22)
#
# Every step runs even if an earlier one fails; the script fails at the end.
# Run from the repository root. Needs a C compiler ($CC, default cc/gcc).

set -u

cd "$(dirname "$0")/../testdata/linktest" || exit 1

out=${TMPDIR:-/tmp}/goffi-linktest.$$
mkdir -p "$out" || exit 1
trap 'rm -rf "$out"' EXIT

failed=""

step() {
	name=$1
	shift
	echo "=> linktest: $name"
	if "$@"; then
		echo "   ok: $name"
	else
		echo "   FAIL: $name"
		failed="$failed [$name]"
	fi
}

# purego's fakecgo, like goffi's, needs -std on FreeBSD with CGO_ENABLED=0.
nocgo_gcflags=""
if [ "$(go env GOOS)" = freebsd ]; then
	nocgo_gcflags="-gcflags=github.com/ebitengine/purego/internal/fakecgo=-std"
fi

go version
go env GOOS GOARCH CC

step "cgo=1 default link" env CGO_ENABLED=1 go run .
step "cgo=1 -linkmode=external" env CGO_ENABLED=1 go run -ldflags=-linkmode=external .
step "cgo=1 -buildmode=c-shared" env CGO_ENABLED=1 go build -buildmode=c-shared -o "$out/liblinktest.so" ./clib
step "cgo=1 -buildmode=c-archive" env CGO_ENABLED=1 go build -buildmode=c-archive -o "$out/liblinktest.a" ./clib
step "cgo=0 -tags nofakecgo (#22)" env CGO_ENABLED=0 go run -tags nofakecgo $nocgo_gcflags .

if [ -n "$failed" ]; then
	echo "linktest FAIL:$failed"
	exit 1
fi
echo "linktest PASS: all link modes"
