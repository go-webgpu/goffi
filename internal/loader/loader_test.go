// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package loader

import (
	"strings"
	"testing"
)

func TestKindString(t *testing.T) {
	for k, want := range map[Kind]string{KindGlibc: "glibc", KindMusl: "musl", KindUnknown: "unknown"} {
		if got := k.String(); got != want {
			t.Errorf("Kind(%d).String() = %q, want %q", k, got, want)
		}
	}
}

func TestDetectConsistent(t *testing.T) {
	e := Detect()
	if e.Kind == KindUnknown {
		return // exotic host or non-target arch; nothing more to assert
	}
	if e.Loader == "" || e.LibC == "" || e.Preload == "" {
		t.Fatalf("Detect returned kind %v with empty Loader/LibC/Preload: %+v", e.Kind, e)
	}
	// The libc itself comes first in the preload list: it is the one object
	// every flavor needs, and the rest only add to it.
	if !strings.HasPrefix(e.Preload+" ", e.LibC+" ") {
		t.Errorf("Preload %q does not start with LibC %q", e.Preload, e.LibC)
	}
	if !fileExists(e.Loader) {
		t.Errorf("Detect chose %q but it does not exist", e.Loader)
	}
}
