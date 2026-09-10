#!/usr/bin/env bash
set -euo pipefail

# Verify the goffi_musl build mode against a real musl userland.
#
# A default goffi binary cannot start on Alpine for two independent reasons:
# PT_INTERP names the glibc loader (execve fails with a misleading ENOENT
# about the binary itself), and DT_NEEDED names glibc SONAMEs that musl does
# not ship. The goffi_musl tag fixes both. This script checks the artifacts
# statically, then executes the probe (cmd/musl-probe) inside an Alpine
# userland, picking the strongest execution mechanism available:
#
#   1. docker run alpine        (CI runners) - kernel resolves PT_INTERP
#   2. chroot into a minirootfs (root)       - kernel resolves PT_INTERP
#   3. ld-musl invoked directly (fallback)   - bypasses PT_INTERP, still
#                                              runs every musl code path
#
# The probe covers each directive group the tag replaces: dlopen/dlsym,
# integer and floating-point calls, errno capture, C-to-Go callbacks, and a
# goroutine hammer that forces the runtime to create OS threads through
# fakecgo's pthread imports. See docs/MUSL.md.

ALPINE_IMAGE=alpine:3.24
ROOTFS_VERSION=3.24.1
ROOTFS_URL="https://dl-cdn.alpinelinux.org/alpine/v3.24/releases/x86_64/alpine-minirootfs-${ROOTFS_VERSION}-x86_64.tar.gz"
ROOTFS_SHA256=41f73e3cf5fa919b8aa5ca6b30dc48f0da2720776d7423e2a7748211456fe081

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

export CGO_ENABLED=0
MUSL_FLAGS=(-tags goffi_musl -gcflags=github.com/go-webgpu/goffi/internal/dl=-std)

echo "==> Compiling with -tags goffi_musl"
for arch in amd64 arm64; do
	if GOOS=linux GOARCH="$arch" go build "${MUSL_FLAGS[@]}" ./...; then
		echo "  ok   linux/${arch}"
	else
		echo "  FAIL linux/${arch}" >&2
		exit 1
	fi
done

echo "==> Verifying interpreter and SONAMEs (amd64 and arm64)"
go test -run TestMuslLinkArtifacts -v ./ffi

echo "==> Building the runtime probe"
probe=$(mktemp -d)
trap 'rm -rf "$probe"' EXIT
GOOS=linux GOARCH=amd64 go build "${MUSL_FLAGS[@]}" -o "$probe/musl-probe" ./cmd/musl-probe

run_probe() {
	if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
		echo "==> Running probe in ${ALPINE_IMAGE} (docker)"
		docker run --rm -v "$probe:/p:ro" "$ALPINE_IMAGE" /p/musl-probe
		return
	fi

	echo "==> No docker; fetching Alpine minirootfs ${ROOTFS_VERSION}"
	curl -fsSL "$ROOTFS_URL" -o "$probe/rootfs.tar.gz"
	echo "${ROOTFS_SHA256}  $probe/rootfs.tar.gz" | sha256sum -c -
	mkdir -p "$probe/rootfs"
	tar xzf "$probe/rootfs.tar.gz" -C "$probe/rootfs"

	if [ "$(id -u)" = 0 ]; then
		echo "==> Running probe via chroot (kernel resolves PT_INTERP)"
		cp "$probe/musl-probe" "$probe/rootfs/musl-probe"
		chroot "$probe/rootfs" /musl-probe
	else
		echo "==> Running probe via ld-musl directly (no root)"
		LD_LIBRARY_PATH="$probe/rootfs/lib" \
			"$probe/rootfs/lib/ld-musl-x86_64.so.1" "$probe/musl-probe"
	fi
}

run_probe

echo "==> musl build mode OK"
