// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build android && arm64

// Android/Bionic dynamic-loader declarations.
package dl

// Android exposes the POSIX dlfcn entry points from libdl.so. It does not
// provide glibc's libdl.so.2 soname, and RTLD_NODELETE is the supported way to
// keep function pointers valid when the caller retains them for the process
// lifetime.

const (
	RTLD_LAZY     = 0x00001
	RTLD_NOW      = 0x00002
	RTLD_GLOBAL   = 0x00100
	RTLD_LOCAL    = 0x00000
	RTLD_NODELETE = 0x01000
)

// RTLD_DEFAULT is Android's default lookup pseudo-handle.
const RTLD_DEFAULT = 0x00000
