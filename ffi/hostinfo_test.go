// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi

import (
	"path/filepath"
	"strings"
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

// TestHostPreload pins the relation between the preload list and the libc:
// empty together, and the libc named first when there is one.
func TestHostPreload(t *testing.T) {
	libc, preload := HostLibC(), HostPreload()
	if libc == "" {
		if preload != "" {
			t.Errorf("HostLibC() is empty but HostPreload() = %q", preload)
		}
		return
	}
	if !strings.HasPrefix(preload+" ", libc+" ") {
		t.Errorf("HostPreload() = %q, want it to start with HostLibC() %q", preload, libc)
	}
	if LibcKind() == "glibc" {
		for _, lib := range []string{"libpthread.so.0", "libdl.so.2"} {
			if !strings.Contains(" "+preload+" ", " "+lib+" ") {
				t.Errorf("HostPreload() = %q on glibc, want it to name %s", preload, lib)
			}
		}
	}
}
