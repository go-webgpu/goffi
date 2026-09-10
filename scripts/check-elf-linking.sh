#!/usr/bin/env bash
# check-elf-linking.sh — assert ELF linking shape for goffi builds.
#
# Usage:
#   scripts/check-elf-linking.sh --dynamic <binary>
#   scripts/check-elf-linking.sh --static <binary>
#
# Exit 0 on success. Requires readelf or llvm-readelf (or Python 3 fallback).

set -euo pipefail

usage() {
	cat <<'EOF'
Usage: scripts/check-elf-linking.sh --dynamic|--static <binary>

  --dynamic   Expect PT_INTERP and at least one DT_NEEDED (default FFI profile)
  --static    Expect no PT_INTERP and no DT_NEEDED (-tags goffi_static)
EOF
	exit 2
}

mode=""
binary=""
while [[ $# -gt 0 ]]; do
	case "$1" in
		--dynamic|--static) mode=${1#--}; shift ;;
		-h|--help) usage ;;
		*) binary=$1; shift ;;
	esac
done
[[ -n "$mode" && -n "$binary" && -f "$binary" ]] || usage

has_interp=0
needed_count=0

if command -v llvm-readelf >/dev/null 2>&1; then
	READELF=(llvm-readelf)
elif command -v readelf >/dev/null 2>&1; then
	READELF=(readelf)
else
	READELF=()
fi

if [[ ${#READELF[@]} -gt 0 ]]; then
	if "${READELF[@]}" -l "$binary" 2>/dev/null | grep -q 'INTERP'; then
		has_interp=1
	fi
	needed_count=$("${READELF[@]}" -d "$binary" 2>/dev/null | grep -c 'NEEDED' || true)
else
	# Portable ELF64 little-endian probe (CI macOS runners without binutils).
	eval "$(python3 - "$binary" <<'PY'
import struct, sys
path = sys.argv[1]
data = open(path, "rb").read()
if data[:4] != b"\x7fELF":
    print("echo 'not ELF'; exit 1")
    raise SystemExit
e_phoff = struct.unpack_from("<Q", data, 32)[0]
e_phentsize = struct.unpack_from("<H", data, 54)[0]
e_phnum = struct.unpack_from("<H", data, 56)[0]
has_interp = 0
needed = 0
dyn_off = dyn_sz = None
for i in range(e_phnum):
    off = e_phoff + i * e_phentsize
    p_type = struct.unpack_from("<I", data, off)[0]
    if p_type == 3:
        has_interp = 1
    if p_type == 2:
        dyn_off = struct.unpack_from("<Q", data, off + 8)[0]
        dyn_sz = struct.unpack_from("<Q", data, off + 32)[0]
if dyn_off is not None:
    pos = dyn_off
    end = dyn_off + dyn_sz
    while pos < end:
        tag, _ = struct.unpack_from("<QQ", data, pos)
        pos += 16
        if tag == 0:
            break
        if tag == 1:
            needed += 1
print(f"has_interp={has_interp}")
print(f"needed_count={needed}")
PY
)"
fi

echo "binary=$binary mode=$mode INTERP=$has_interp NEEDED=$needed_count"

if [[ "$mode" == "static" ]]; then
	if [[ "$has_interp" -ne 0 || "$needed_count" -ne 0 ]]; then
		echo "ERROR: expected fully static ELF (no PT_INTERP, no DT_NEEDED)" >&2
		exit 1
	fi
	echo "OK: statically linked"
else
	if [[ "$has_interp" -eq 0 ]]; then
		echo "ERROR: expected PT_INTERP for dynamic FFI profile" >&2
		exit 1
	fi
	if [[ "$needed_count" -eq 0 ]]; then
		echo "ERROR: expected DT_NEEDED entries for dynamic FFI profile" >&2
		exit 1
	fi
	echo "OK: dynamically linked with NEEDED libs"
fi
