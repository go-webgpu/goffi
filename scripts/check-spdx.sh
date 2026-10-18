#!/bin/sh
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors
#
# check-spdx.sh — keep license headers true to where the code came from.
#
# goffi is MIT. A file may carry an Apache-2.0 SPDX header only when it is
# derived from ebitengine/purego (Apache-2.0), and then it must keep
# purego's copyright line ("The Ebitengine Authors"). An Apache header
# without that line marks original code that was given the wrong license,
# usually by copying a neighbouring file's header.
#
# PENDING lists original files written by contributors under an Apache
# header. Changing their license needs each author's consent; remove a file
# from the list when its header is changed to MIT.
#
# Usage: scripts/check-spdx.sh   (from the repository root)

set -eu

PENDING='
ffi/callback_android_arm64.go
ffi/callback_android_arm64_test.go
ffi/dl_android.go
ffi/float_return_windows_amd64_test.go
ffi/struct_e2e_test.go
internal/arch/arm64/fakecgo_test.go
internal/dl/dl_android.go
internal/dl/dl_android_cgo.go
internal/dl/dl_android_nocgo.go
internal/dl/dl_android_nocgo_arm64.s
internal/fakecgo/android_abi_test.go
internal/fakecgo/android_dl.go
internal/fakecgo/android_dl_stubs_arm64.s
internal/fakecgo/libcgo_android.go
internal/syscall/cgo.go
internal/syscall/errno_android.go
internal/syscall/errno_android_cgo.go
'

apache='SPDX-License-Identifier: Apache-2.0'

# header prints the first lines of a file, where SPDX tags live.
header() { head -n 6 "$1" 2>/dev/null; }

bad=0
for f in $(git ls-files); do
	header "$f" | grep -q "$apache" || continue
	header "$f" | grep -q 'The Ebitengine Authors' && continue
	case "$PENDING" in
	*"
$f
"*) continue ;;
	esac
	echo "$f: Apache-2.0 header without purego's copyright line; original code is MIT"
	bad=1
done

# A file that got its MIT header must leave the list, so the list never
# hides a new file.
for f in $PENDING; do
	if ! header "$f" | grep -q "$apache"; then
		echo "$f: in PENDING but no longer Apache-2.0 (or gone); remove it from the list"
		bad=1
	fi
done

if [ "$bad" -ne 0 ]; then
	echo
	echo "Use the MIT header for original code:"
	echo "  // SPDX-License-Identifier: MIT"
	echo "  // SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors"
	exit 1
fi
echo "SPDX headers OK"
