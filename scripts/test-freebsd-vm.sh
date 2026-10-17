#!/bin/sh
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors
#
# test-freebsd-vm.sh — run goffi's test suite on a real FreeBSD system.
#
# CI runs this inside a FreeBSD VM (vmactions/freebsd-vm, see the
# test-freebsd job in .github/workflows/ci.yml). Cross-compiling for FreeBSD
# proves the code builds; only running it proves the call path, callbacks and
# internal/fakecgo work against FreeBSD's libc, libthr and kernel.
#
# Requires Go in /usr/local/go and the base system C compiler (cc).

set -eu

PATH=/usr/local/go/bin:$PATH
export PATH

# The C-backed end-to-end tests build ffi/testdata/structtest.c with $CC
# (default gcc, which FreeBSD's base system does not ship). Use the system
# compiler and fail, instead of skipping, if the library cannot be built.
CC=cc
GOFFI_REQUIRE_TESTLIB=1
export CC GOFFI_REQUIRE_TESTLIB

echo "=> system"
uname -a
go version
cc --version
go env GOOS GOARCH CC

# Run both modes even when the first one fails, so one CI run shows both;
# the script fails at the end if either did.
failed=""

# With CGO_ENABLED=0, internal/fakecgo must be compiled as a std package so
# that its //go:cgo_export_dynamic directives are accepted on FreeBSD.
echo "=> go test CGO_ENABLED=0"
CGO_ENABLED=0 go test -count=1 -v \
	-gcflags=github.com/go-webgpu/goffi/internal/fakecgo=-std ./... ||
	failed="$failed CGO_ENABLED=0"

echo "=> go test CGO_ENABLED=1"
CGO_ENABLED=1 go test -count=1 -v ./... ||
	failed="$failed CGO_ENABLED=1"

# goffi's tests alone link internally; other cgo code, c-shared and
# c-archive force external linking (#72). Cover both explicitly.
echo "=> go test CGO_ENABLED=1 -linkmode=external"
CGO_ENABLED=1 go test -count=1 -ldflags=-linkmode=external ./... ||
	failed="$failed CGO_ENABLED=1-external"

echo "=> go test CGO_ENABLED=1 -linkmode=internal"
CGO_ENABLED=1 go test -count=1 -ldflags=-linkmode=internal ./... ||
	failed="$failed CGO_ENABLED=1-internal"

# goffi next to purego and an ordinary cgo package, in every link mode.
echo "=> linktest"
sh scripts/run-linktest.sh ||
	failed="$failed linktest"

if [ -n "$failed" ]; then
	echo "FAIL:$failed"
	exit 1
fi
echo "PASS: CGO_ENABLED=0, CGO_ENABLED=1 (default, external, internal linking) and linktest"
