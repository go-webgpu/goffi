// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi

import (
	"path/filepath"
	"testing"
)

// TestHostInfo exercises the public host-inspection helpers. The concrete
// values are host-dependent, so the test asserts the documented contract
// rather than a fixed string: LibcKind is always one of the three documented
// flavors, and the loader path and libc SONAME are populated together with a
// recognized libc and empty together with an unknown one.
func TestHostInfo(t *testing.T) {
	loaderPath := HostLoader()
	libc := HostLibC()
	kind := LibcKind()

	switch kind {
	case "glibc", "musl", "unknown":
		// documented set
	default:
		t.Fatalf("LibcKind() = %q, want one of glibc/musl/unknown", kind)
	}

	if kind == "unknown" {
		// Nothing to re-exec through: both strings must be empty so callers
		// can treat "" as "no universal loader available".
		if loaderPath != "" || libc != "" {
			t.Errorf("unknown libc but HostLoader()=%q HostLibC()=%q, want both empty",
				loaderPath, libc)
		}
		return
	}

	// A recognized libc must report a concrete, absolute loader path and a
	// non-empty SONAME: a universal binary re-execs through exactly these.
	if loaderPath == "" {
		t.Errorf("HostLoader() is empty for kind %q", kind)
	} else if !filepath.IsAbs(loaderPath) {
		t.Errorf("HostLoader() = %q, want an absolute path", loaderPath)
	}
	if libc == "" {
		t.Errorf("HostLibC() is empty for kind %q", kind)
	}
}
