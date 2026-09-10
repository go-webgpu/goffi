// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux && !android && (amd64 || arm64) && !goffi_static && goffi_universal

// __errno_location for the portable "universal" Linux build.
//
// Both glibc and musl export __errno_location under that exact name, so the
// only thing that differs between them is the SONAME the symbol is imported
// from. Importing it with an empty library name (no DT_NEEDED) makes this one
// binary bind __errno_location from whichever libc the host provides, after
// the fakecgo re-exec bridge maps it. See internal/dl/dl_universal.go and
// docs/PROFILE_U.md.

package syscall

//go:cgo_import_dynamic goffi_errno_location __errno_location ""
