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

// HostPreload returns the list of objects a universal binary has the host
// loader preload (`<loader> --preload <list>`), or "" whenever HostLibC does.
// On musl it is HostLibC alone. On glibc it is libc.so.6 followed by
// libpthread.so.0 and libdl.so.2: before glibc 2.34 the pthread_* and dl*
// functions a universal binary imports live there rather than in libc.so.6.
// A program that starts another copy of itself through the host loader passes
// this, not HostLibC, exactly as the re-exec bridge does.
func HostPreload() string { return loader.Detect().Preload }

// LibcKind returns the host libc flavor as "glibc", "musl", or "unknown".
func LibcKind() string { return loader.Detect().Kind.String() }
