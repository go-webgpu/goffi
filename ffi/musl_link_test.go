// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux && !android && (amd64 || arm64)

package ffi_test

import (
	"bytes"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// muslNames maps GOARCH to the musl architecture that appears in both the
// loader path and the libc SONAME.
var muslNames = map[string]string{
	"amd64": "x86_64",
	"arm64": "aarch64",
}

// TestMuslLinkArtifacts is the link-time half of the goffi_musl verification
// (the runtime half is cmd/musl-probe, driven by scripts/check-musl.sh).
//
// A goffi binary is unusable on Alpine for two independent reasons, and each
// gets its own assertion here: the glibc SONAMEs in DT_NEEDED (musl ships no
// libdl.so.2, libc.so.6 or libpthread.so.0, so the dynamic linker refuses to
// start the process), and PT_INTERP, which the Go linker defaults to the
// glibc loader path (so on Alpine execve fails before a single instruction
// runs). The goffi_musl tag must fix both, on both architectures.
func TestMuslLinkArtifacts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: builds two binaries")
	}

	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("locate module root: %v", err)
	}

	for goarch, musl := range muslNames {
		t.Run(goarch, func(t *testing.T) {
			bin := filepath.Join(t.TempDir(), "musl-probe-"+goarch)
			buildMuslProbe(t, root, goarch, bin)

			f, err := elf.Open(bin)
			if err != nil {
				t.Fatalf("open ELF: %v", err)
			}
			defer f.Close()

			wantInterp := "/lib/ld-musl-" + musl + ".so.1"
			interp := readInterp(t, f)
			if interp != wantInterp {
				t.Errorf("PT_INTERP = %q, want %q", interp, wantInterp)
			}

			wantLibc := "libc.musl-" + musl + ".so.1"
			needed, err := f.DynString(elf.DT_NEEDED)
			if err != nil {
				t.Fatalf("read DT_NEEDED: %v", err)
			}
			if len(needed) != 1 || needed[0] != wantLibc {
				t.Errorf("DT_NEEDED = %v, want exactly [%s]", needed, wantLibc)
			}
			for _, glibc := range []string{"libdl.so.2", "libc.so.6", "libpthread.so.0"} {
				for _, n := range needed {
					if n == glibc {
						t.Errorf("glibc SONAME %s leaked into the musl build", glibc)
					}
				}
			}
		})
	}
}

func readInterp(t *testing.T, f *elf.File) string {
	t.Helper()
	sec := f.Section(".interp")
	if sec == nil {
		t.Fatal("binary has no .interp section")
	}
	data, err := sec.Data()
	if err != nil {
		t.Fatalf("read .interp: %v", err)
	}
	return string(bytes.TrimRight(data, "\x00"))
}

func buildMuslProbe(t *testing.T, root, goarch, out string) {
	t.Helper()

	goTool := goToolPath()

	cmd := exec.Command(goTool, "build",
		"-tags", "goffi_musl",
		// //go:cgo_dynamic_linker is restricted to cgo-generated code; the
		// musl interpreter directive lives in internal/dl, so that one
		// package is compiled with the check relaxed.
		"-gcflags=github.com/go-webgpu/goffi/internal/dl=-std",
		"-o", out, "./cmd/musl-probe")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH="+goarch,
		"GOFLAGS=-mod=mod",
	)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build linux/%s with -tags goffi_musl: %v\n%s", goarch, err, outBytes)
	}
}
