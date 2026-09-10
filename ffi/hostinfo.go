// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi

import "github.com/go-webgpu/goffi/internal/loader"

// HostLoader returns the absolute path of the host's dynamic loader (ld.so) for
// the running libc flavor, or "" if it cannot be determined (an unsupported
// architecture, or a host running neither glibc nor musl). This is the loader a
// universal binary re-execs through.
func HostLoader() string { return loader.Detect().Loader }

// HostLibC returns the host libc SONAME (e.g. "libc.so.6" on glibc or
// "libc.musl-x86_64.so.1" on musl), or "" if it cannot be determined.
func HostLibC() string { return loader.Detect().LibC }

// LibcKind returns the host libc flavor as "glibc", "musl", or "unknown".
func LibcKind() string { return loader.Detect().Kind.String() }
