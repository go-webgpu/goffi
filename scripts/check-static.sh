#!/usr/bin/env bash
set -euo pipefail

# Verify the goffi_static build mode.
#
# Without the tag, goffi's //go:cgo_import_dynamic directives force the Go
# linker to emit a PT_INTERP header and DT_NEEDED entries even under
# CGO_ENABLED=0, so every binary importing goffi is dynamically linked and will
# not start on Alpine or in a scratch container. -extldflags '-static' cannot
# undo that: no external linker is involved. With -tags goffi_static those
# directives are gone and the linker produces a static executable.
#
# This script checks that (a) the tagged build still compiles on every
# platform, (b) the linker output really has no interpreter and no shared
# library dependencies, and (c) the tag stays a no-op on the platforms that are
# always dynamic (Windows, Android). See unxed/f4#693.

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

export CGO_ENABLED=0

echo "==> Compiling with -tags goffi_static"
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 \
	windows/amd64 windows/arm64 android/arm64; do
	os="${target%%/*}"
	arch="${target##*/}"
	if GOOS="$os" GOARCH="$arch" go build -tags goffi_static ./...; then
		echo "  ok   ${target}"
	else
		echo "  FAIL ${target}" >&2
		exit 1
	fi
done

# FreeBSD needs the same -gcflags="-std" workaround as the cross-compile job:
# internal/fakecgo is not imported in a static build, but `go build ./...`
# still compiles the package, and its //go:cgo_export_dynamic directives are
# rejected outside cgo-generated code.
for arch in amd64 arm64; do
	if GOOS=freebsd GOARCH="$arch" go build -tags goffi_static \
		-gcflags="github.com/go-webgpu/goffi/internal/fakecgo=-std" ./...; then
		echo "  ok   freebsd/${arch}"
	else
		echo "  FAIL freebsd/${arch}" >&2
		exit 1
	fi
done

echo "==> Verifying linker output for linux/amd64 and linux/arm64"
# The test builds a real consumer binary and inspects it with debug/elf, so no
# binutils are needed on the runner.
go test -run TestStaticBuildProducesStaticBinary -v ./ffi

echo "==> Checking that the tag is a no-op where it must be"
# Windows and Android are always dynamically linked (kernel32 / Bionic), so a
# binary built with the tag must keep full FFI there. Available() is a
# compile-time constant, so a failed assertion is a build failure.
probe=$(mktemp -d)
trap 'rm -rf "$probe"' EXIT
cat >"$probe/main.go" <<'PROBE'
package main

import "github.com/go-webgpu/goffi/ffi"

func main() {
	if !ffi.Available() {
		panic("goffi_static must not disable FFI on this platform")
	}
}
PROBE
cat >"$probe/go.mod" <<PROBE
module goffistaticprobe

go 1.26.0

require github.com/go-webgpu/goffi v0.0.0

replace github.com/go-webgpu/goffi => $ROOT
PROBE
for target in windows/amd64 android/arm64; do
	os="${target%%/*}"
	arch="${target##*/}"
	if (cd "$probe" && GOOS="$os" GOARCH="$arch" GOFLAGS=-mod=mod \
		go build -tags goffi_static -o /dev/null .); then
		echo "  ok   ${target} keeps FFI"
	else
		echo "  FAIL ${target} lost FFI" >&2
		exit 1
	fi
done

echo "==> Static build mode OK"
