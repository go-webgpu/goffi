// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build amd64 && (linux || darwin || freebsd) && !cgo && !nofakecgo

package amd64

// The test binary of this package links internal/syscall, which imports
// errno helpers from the C library (__error from libc.so.7 on FreeBSD). With
// CGO_ENABLED=0, a program using the C library needs internal/fakecgo, as the
// ffi package does (ffi/fakecgo_unix.go): it supplies runtime.cgocall and, on
// FreeBSD, the environ and __progname symbols that libc.so.7 expects the
// executable to define. Without this import, FreeBSD's dynamic linker refuses
// to start the test binary:
//
//	ld-elf.so.1: /lib/libc.so.7: Undefined symbol "environ"
//
// The conditions match ffi/fakecgo_unix.go; with CGO_ENABLED=1 runtime/cgo
// provides all of this, and with nofakecgo another package does.
import _ "github.com/go-webgpu/goffi/internal/fakecgo"
