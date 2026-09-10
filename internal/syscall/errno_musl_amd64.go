// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux && !android && !goffi_static && goffi_musl && !goffi_universal && amd64

// musl flavor of errno_linux.go: same symbol, different SONAME. musl
// exports __errno_location from its single libc object (errno itself lives
// in the thread control block, but the accessor is a plain exported
// function), so only the library name changes.

package syscall

//go:cgo_import_dynamic goffi_errno_location __errno_location "libc.musl-x86_64.so.1"
//go:cgo_import_dynamic _ _ "libc.musl-x86_64.so.1"
