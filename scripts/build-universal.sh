#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors
#
# build-universal.sh -- build a portable "universal" goffi binary that runs
# FFI on both glibc and musl systems from a single artifact.
#
# It builds with -tags goffi_universal and CGO_ENABLED=0 (the whole point is a
# fully portable, C-toolchain-free binary), then strips the ELF interpreter so
# the kernel loads the binary directly everywhere. At startup goffi re-execs
# through the host's own dynamic loader with the host libc pre-loaded; see
# docs/PROFILE_U.md.
#
# Usage:
#   scripts/build-universal.sh -o <output> <package> [extra go build args...]
#
# Example:
#   scripts/build-universal.sh -o /tmp/uprobe ./cmd/universal-probe
set -euo pipefail

out=""
args=()
while [[ $# -gt 0 ]]; do
	case "$1" in
	-o)
		out="$2"
		shift 2
		;;
	*)
		args+=("$1")
		shift
		;;
	esac
done

if [[ -z "$out" || ${#args[@]} -eq 0 ]]; then
	echo "usage: $0 -o <output> <package> [go build args...]" >&2
	exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> building $out (goffi_universal, CGO_ENABLED=0)"
CGO_ENABLED=0 go build -tags goffi_universal -o "$out" "${args[@]}"

echo "==> stripping PT_INTERP"
go run "$repo_root/cmd/goffi-strip-interp" "$out"

echo "==> done: $out"
echo "    Verify the Profile U contract with: go run ./cmd/goffi-audit $out"
