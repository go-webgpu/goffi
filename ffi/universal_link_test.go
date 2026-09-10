// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux && (amd64 || arm64)

package ffi

import (
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestUniversalNoDTNeeded builds a CGO-free -tags goffi_universal binary and
// asserts it carries no DT_NEEDED entries (the empty-SONAME imports must not
// pull in a specific libc). The PT_INTERP strip is a separate build step
// (scripts/build-universal.sh) checked by cmd/goffi-audit, so it is not
// asserted here.
func TestUniversalNoDTNeeded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: builds a binary")
	}
	out := filepath.Join(t.TempDir(), "universal-probe")
	cmd := exec.Command("go", "build", "-tags", "goffi_universal",
		"-o", out, "github.com/go-webgpu/goffi/cmd/universal-probe")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build universal binary: %v\n%s", err, b)
	}
	f, err := elf.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	libs, err := f.ImportedLibraries()
	if err != nil {
		t.Fatal(err)
	}
	if len(libs) > 0 {
		t.Errorf("universal binary has DT_NEEDED %v, want none", libs)
	}
}
