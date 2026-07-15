// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build android && !cgo && arm64

package syscall

// Bionic exports __errno from libc.so. __errno_location is a glibc name and
// must never appear in an Android artifact.
//
//go:cgo_import_dynamic goffi_errno_location __errno "libc.so"
//go:cgo_import_dynamic _ _ "libc.so"
