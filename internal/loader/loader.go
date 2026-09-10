// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// Package loader is the auditable "Profile U" loader table: for each supported
// architecture it records the host dynamic loader path and libc SONAME of the
// two libc flavors goffi's universal build targets (glibc and musl), and probes
// the running host to report which is present.
//
// This mirrors the constants used by the universal re-exec bridge
// (internal/fakecgo/reexec_table_*.go); a test asserts the two stay in sync.
// Unlike the bridge, this package is plain Go usable from any build mode, and
// backs the public ffi.HostLoader / ffi.HostLibC / ffi.LibcKind helpers.
package loader

import "os"

// Kind identifies a libc flavor.
type Kind int

const (
	KindUnknown Kind = iota
	KindGlibc
	KindMusl
)

// String returns "glibc", "musl", or "unknown".
func (k Kind) String() string {
	switch k {
	case KindGlibc:
		return "glibc"
	case KindMusl:
		return "musl"
	default:
		return "unknown"
	}
}

// Entry describes one libc: its dynamic loader path and its libc SONAME.
type Entry struct {
	Loader string // absolute path to the dynamic loader (ld.so)
	LibC   string // libc SONAME, passed bare to `<loader> --preload <soname>`
	Kind   Kind
}

// Detect reports the libc flavor of the running host by probing the known
// loader paths (glibc first, then musl), matching the re-exec bridge's choice.
// It returns an Entry with Kind == KindUnknown on architectures goffi's
// universal build does not target, or when neither loader is present.
func Detect() Entry {
	if !Known {
		return Entry{Kind: KindUnknown}
	}
	if fileExists(Glibc.Loader) {
		return Glibc
	}
	if fileExists(Musl.Loader) {
		return Musl
	}
	return Entry{Kind: KindUnknown}
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}
