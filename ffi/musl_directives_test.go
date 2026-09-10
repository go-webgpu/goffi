// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// The musl directive files mirror hand-picked glibc originals, and the two
// must not drift: a symbol added to the glibc side and forgotten on the musl
// side would fail only at load time on Alpine, far from the change that
// caused it. This test pins the invariant at go-test time.
//
// One asymmetry is intentional and encoded below: musl's dynamic linker
// binds every import immediately at load and aborts on an unresolved one,
// so pthread_get_stacksize_np -- a Darwin-only API that glibc's lazy PLT
// silently tolerates -- must be absent from the musl set.

var importDirective = regexp.MustCompile(
	`(?m)^//go:cgo_import_dynamic\s+(\S+)\s+(\S+)\s+"([^"]+)"`)

var interpDirective = regexp.MustCompile(
	`(?m)^//go:cgo_dynamic_linker\s+"([^"]+)"`)

// symbolSet returns the imported C symbol names in a file, skipping the
// "_ _" force-dependency entries, plus the set of SONAMEs referenced.
func symbolSet(t *testing.T, path string) (map[string]bool, map[string]bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	syms := map[string]bool{}
	sos := map[string]bool{}
	for _, m := range importDirective.FindAllStringSubmatch(string(data), -1) {
		sos[m[3]] = true
		if m[2] == "_" {
			continue
		}
		syms[m[2]] = true
	}
	return syms, sos
}

func TestMuslDirectiveParity(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("locate module root: %v", err)
	}
	join := func(elem ...string) string {
		return filepath.Join(append([]string{root}, elem...)...)
	}

	muslArches := map[string]string{
		"amd64": "x86_64",
		"arm64": "aarch64",
	}

	// Package -> glibc source of truth and the symbols musl must not carry.
	cases := []struct {
		name    string
		glibc   string
		muslFmt string // per-arch musl file, %s = goarch
		exclude map[string]bool
	}{
		{
			name:    "fakecgo",
			glibc:   join("internal", "fakecgo", "symbols_linux.go"),
			muslFmt: join("internal", "fakecgo", "symbols_musl_%s.go"),
			exclude: map[string]bool{"pthread_get_stacksize_np": true},
		},
		{
			name:    "dl",
			glibc:   join("internal", "dl", "dl_linux_dynamic.go"),
			muslFmt: join("internal", "dl", "dl_musl_%s.go"),
		},
		{
			name:    "syscall",
			glibc:   join("internal", "syscall", "errno_linux.go"),
			muslFmt: join("internal", "syscall", "errno_musl_%s.go"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want, _ := symbolSet(t, tc.glibc)
			for s := range tc.exclude {
				if !want[s] {
					t.Errorf("exclusion list mentions %s, but the glibc file does not import it; update this test", s)
				}
				delete(want, s)
			}

			for goarch, musl := range muslArches {
				path := fmt.Sprintf(tc.muslFmt, goarch)
				got, sos := symbolSet(t, path)

				for s := range want {
					if !got[s] {
						t.Errorf("%s: glibc imports %s but the musl file does not", path, s)
					}
				}
				for s := range got {
					if !want[s] {
						t.Errorf("%s: imports %s, which the glibc file does not (or which musl must not import)", path, s)
					}
				}

				wantSO := "libc.musl-" + musl + ".so.1"
				for so := range sos {
					if so != wantSO {
						t.Errorf("%s: imports from %q, want only %q", path, so, wantSO)
					}
				}
			}
		})
	}

	// The interpreter directive lives in internal/dl and must name the
	// matching musl loader on each architecture.
	for goarch, musl := range muslArches {
		path := join("internal", "dl", fmt.Sprintf("dl_musl_%s.go", goarch))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		m := interpDirective.FindStringSubmatch(string(data))
		want := "/lib/ld-musl-" + musl + ".so.1"
		if m == nil {
			t.Errorf("%s: missing //go:cgo_dynamic_linker directive", path)
		} else if m[1] != want {
			t.Errorf("%s: interpreter %q, want %q", path, m[1], want)
		}
	}
}
