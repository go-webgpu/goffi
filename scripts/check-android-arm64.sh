#!/usr/bin/env bash
set -euo pipefail

# Cross-build and inspect Android arm64/API 29+ artifacts. The script is
# intentionally device-free: compile-time Go/C ABI checks and ELF dependency
# checks catch the portability regressions that a host runner can observe.

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
NDK=${ANDROID_NDK_HOME:-}
if [[ -z "$NDK" ]]; then
	cat >&2 <<'EOF'
ANDROID_NDK_HOME must point at an Android NDK r29 installation.
EOF
	exit 2
fi

case "$(uname -s)-$(uname -m)" in
	Darwin-arm64|Darwin-x86_64) host_tag=darwin-x86_64 ;;
	Linux-x86_64) host_tag=linux-x86_64 ;;
	*) echo "unsupported NDK host: $(uname -s)-$(uname -m)" >&2; exit 2 ;;
esac

toolchain="$NDK/toolchains/llvm/prebuilt/$host_tag/bin"
cc="$toolchain/aarch64-linux-android29-clang"
readelf="$toolchain/llvm-readelf"
[[ -x "$cc" ]] || { echo "missing Android compiler: $cc" >&2; exit 2; }
[[ -x "$readelf" ]] || { echo "missing llvm-readelf: $readelf" >&2; exit 2; }

# This port deliberately depends on the Android arm64 startup ABI in the Go
# runtime. Fail closed when the selected toolchain is outside the audited
# lines, or when any audited source invariant changes.
go_version=$(go env GOVERSION)
case "$go_version" in
	go1.25.12|go1.26.5) ;;
	*)
		echo "unsupported Go runtime source for Android fakecgo: $go_version" >&2
		echo "audit the new runtime/cgo Android arm64 startup ABI before extending this gate" >&2
		exit 2
		;;
esac

goroot=$(go env GOROOT)
runtime_asm="$goroot/src/runtime/asm_arm64.s"
runtime_tls="$goroot/src/runtime/tls_arm64.s"
runtime_android_cgo="$goroot/src/runtime/cgo/gcc_android.c"

require_source_pattern() {
	local file=$1
	local pattern=$2
	local description=$3
	grep -Eq "$pattern" "$file" || {
		echo "Go $go_version runtime drift: missing $description in $file" >&2
		exit 1
	}
}

android_callsite=$(awk '
	/^#ifdef GOOS_android$/ { capture = 1 }
	capture { print }
	capture && /BL[[:space:]]+\(R12\)/ { exit }
' "$runtime_asm")
for pattern in \
	'MRS_TPIDR_R0' \
	'MOVD[[:space:]]+R0,[[:space:]]*R3' \
	'MOVD[[:space:]]+\$runtime·tls_g\(SB\),[[:space:]]*R2' \
	'MOVD[[:space:]]+\$setg_gcc<>\(SB\),[[:space:]]*R1' \
	'MOVD[[:space:]]+g,[[:space:]]*R0' \
	'BL[[:space:]]+\(R12\)'
do
	grep -Eq "$pattern" <<<"$android_callsite" || {
		echo "Go $go_version runtime drift in Android _cgo_init callsite: $pattern" >&2
		exit 1
	}
done
require_source_pattern "$runtime_tls" 'DATA runtime·tls_g\+0\(SB\)/8,[[:space:]]*\$16' 'API-29 TLS slot-2 offset'
require_source_pattern "$runtime_android_cgo" '^#define TLS_SLOT_APP 2$' 'Bionic TLS_SLOT_APP contract'
require_source_pattern "$runtime_android_cgo" 'dlsym\(handle, "android_get_device_api_level"\)' 'Android API-level startup probe'

# GOOS=android also satisfies linux build constraints. The shared fakecgo
# wiring must therefore remain selected: iscgo=true admits outbound cgocall,
# while the hooks satisfy the runtime paths enabled by that state. Public
# Android callbacks are rejected separately in ffi.
echo "Checking Android fakecgo runtime wiring"
android_fakecgo_files=$(
	GOOS=android GOARCH=arm64 CGO_ENABLED=0 \
		go list -f '{{range .GoFiles}}{{println .}}{{end}}' ./internal/fakecgo
)
for required in callbacks.go iscgo.go setenv.go; do
	grep -Fxq "$required" <<<"$android_fakecgo_files" || {
		echo "Android fakecgo runtime wiring omitted $required" >&2
		exit 1
	}
done

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Checking generated Android fakecgo sources"
generator_dir="$tmp/fakecgo-generator"
mkdir -p "$generator_dir"
cp "$ROOT/internal/fakecgo/gen.go" "$generator_dir/gen.go"
(
	cd "$generator_dir"
	go run gen.go
)
for generated in \
	symbols.go \
	symbols_android.go \
	symbols_android_imports.go \
	symbols_darwin.go \
	symbols_freebsd.go \
	symbols_linux.go \
	symbols_netbsd.go \
	trampolines_stubs.s \
	trampolines_stubs_android.s
do
	cmp "$ROOT/internal/fakecgo/$generated" "$generator_dir/$generated" || {
		echo "generated fakecgo source is stale: $generated" >&2
		exit 1
	}
done

echo "Checking NDK C ABI headers"
"$cc" -std=c11 -Wall -Werror -fsyntax-only "$ROOT/testdata/android_abi_probe.c"

for cgo in 0 1; do
	echo "Building Android arm64 (CGO_ENABLED=$cgo)"
	if [[ "$cgo" == 1 ]]; then
		CC="$cc" GOOS=android GOARCH=arm64 CGO_ENABLED=1 \
			go test -exec=true ./... 2>&1
		CC="$cc" GOOS=android GOARCH=arm64 CGO_ENABLED=1 \
			go vet ./...
		CC="$cc" GOOS=android GOARCH=arm64 CGO_ENABLED=1 \
			go test -c -o "$tmp/ffi-cgo.test" ./ffi
	else
		GOOS=android GOARCH=arm64 CGO_ENABLED=0 \
			go test -exec=true ./... 2>&1
		GOOS=android GOARCH=arm64 CGO_ENABLED=0 \
			go vet ./...
		GOOS=android GOARCH=arm64 CGO_ENABLED=0 \
			go test -c -o "$tmp/ffi-nocgo.test" ./ffi
	fi

	artifact="$tmp/ffi-$([[ "$cgo" == 1 ]] && echo cgo || echo nocgo).test"
	file_header=$("$readelf" -h "$artifact")
	grep -q 'Machine:.*AArch64' <<<"$file_header" || {
		echo "unexpected machine in $artifact" >&2
		exit 1
	}
	dynamic=$("$readelf" -d "$artifact")
	if grep -Eiq 'GLIBC|libpthread|\.so\.2|libc\.so\.6|libdl\.so\.2|__errno_location' <<<"$dynamic"; then
		echo "forbidden non-Bionic dependency in $artifact" >&2
		grep -Ei 'GLIBC|libpthread|\.so\.2|libc\.so\.6|libdl\.so\.2|__errno_location' <<<"$dynamic" >&2
		exit 1
	fi
	dynsyms=$("$readelf" --dyn-syms "$artifact")
	if grep -Eiq '__errno_location|callback|trampoline' <<<"$dynsyms"; then
		echo "forbidden callback/glibc symbol in $artifact" >&2
		grep -Ein '__errno_location|callback|trampoline' <<<"$dynsyms" >&2
		exit 1
	fi
	grep -q 'Shared library: \[libc\.so\]' <<<"$dynamic" || {
		echo "missing Bionic libc dependency in $artifact" >&2
		exit 1
	}
	if [[ "$cgo" == 0 ]]; then
		# The fakecgo startup trampoline must preserve the four AAPCS64 input
		# registers and tail-call the Go implementation indirectly. The Go
		# implementation must enter the API/TLS guard before any ordinary cgo
		# path; fakecgo x_cgo_init must not call runtime.cgocall itself.
		trampoline=$(go tool objdump -s '^x_cgo_init_trampoline$' "$artifact")
		grep -q 'CALL (R9)' <<<"$trampoline" || {
			echo "x_cgo_init trampoline is not an indirect AAPCS64 call" >&2
			exit 1
		}
		if awk '
			/MOVD/ {
				line = $0
				gsub(/[^[:alnum:]]/, " ", line)
				count = split(line, fields)
				for (i = 1; i <= count; i++) {
					if (fields[i] ~ /^R[0-3]$/) found = 1
				}
			}
			END { exit found ? 0 : 1 }
		' <<<"$trampoline"; then
			echo "x_cgo_init trampoline clobbers an AAPCS64 input register" >&2
			exit 1
		fi
		startup=$(go tool objdump -s '^github.com/go-webgpu/goffi/internal/fakecgo.x_cgo_init$' "$artifact")
		grep -q 'CALL github.com/go-webgpu/goffi/internal/fakecgo.x_cgo_inittls' <<<"$startup" || {
			echo "x_cgo_init does not enter the Android TLS guard" >&2
			exit 1
		}
		guard_line=$(grep -n 'CALL github.com/go-webgpu/goffi/internal/fakecgo.x_cgo_inittls' <<<"$startup" | head -n1 | cut -d: -f1)
		call5_line=$(grep -n 'CALL github.com/go-webgpu/goffi/internal/fakecgo.call5.abi0' <<<"$startup" | head -n1 | cut -d: -f1 || true)
		if [[ -n "$call5_line" && "$guard_line" -ge "$call5_line" ]]; then
			echo "x_cgo_init reaches a Bionic call before the Android TLS guard" >&2
			exit 1
		fi
		if grep -q 'runtime.cgocall' <<<"$startup"; then
			echo "x_cgo_init reached runtime.cgocall before startup guard" >&2
			exit 1
		fi

		# Loader failures must capture and duplicate dlerror before the one
		# runtime.cgocall returns; dlerror state is local to that OS thread.
		open_wrapper=$(go tool objdump -s '^androidDlopenWrapper$' "$artifact")
		sym_wrapper=$(go tool objdump -s '^androidDlsymWrapper$' "$artifact")
		open_calls=$(grep -c 'CALL (R10)' <<<"$open_wrapper")
		sym_calls=$(grep -c 'CALL (R10)' <<<"$sym_wrapper")
		if [[ "$open_calls" -ne 3 || "$sym_calls" -ne 4 ]]; then
			echo "Android loader wrapper no longer captures dlerror in-call" >&2
			exit 1
		fi
	fi
	done

echo "Android arm64/API 29+ checks passed"
