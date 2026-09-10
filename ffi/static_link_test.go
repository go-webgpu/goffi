// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux && !android && (amd64 || arm64)

package ffi_test

import (
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// probeSource is a throwaway program that links the ffi package the way a real
// consumer does. Building it from a test is the only way to observe what the
// linker emits: the test binary itself is always dynamic, because the test
// harness links the dynamic build of this package.
const probeSource = `package main

import (
	"errors"
	"fmt"

	"github.com/go-webgpu/goffi/ffi"
)

func main() {
	fmt.Printf("available=%v\n", ffi.Available())
	_, err := ffi.LoadLibrary("libm.so.6")
	fmt.Printf("static_err=%v\n", errors.Is(err, ffi.ErrStaticBuild))
}
`

// TestStaticBuildProducesStaticBinary is the regression test for unxed/f4#693.
//
// Any //go:cgo_import_dynamic directive reachable from the binary makes the Go
// linker emit a PT_INTERP header and DT_NEEDED entries, even under
// CGO_ENABLED=0, and -extldflags '-static' cannot undo it because no external
// linker runs. The goffi_static tag must remove every one of those directives
// for both linux/amd64 and linux/arm64, otherwise the binary will not start on
// Alpine or in a scratch container.
func TestStaticBuildProducesStaticBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: builds two binaries")
	}

	dir := writeProbeModule(t)

	for _, goarch := range []string{"amd64", "arm64"} {
		t.Run(goarch, func(t *testing.T) {
			bin := filepath.Join(dir, "probe-"+goarch)
			build(t, dir, goarch, bin)

			f, err := elf.Open(bin)
			if err != nil {
				t.Fatalf("open ELF: %v", err)
			}
			defer f.Close()

			for _, p := range f.Progs {
				if p.Type == elf.PT_INTERP {
					t.Error("binary has a PT_INTERP header: it still requires an ELF interpreter")
				}
			}

			// DynamicSection returns an error when there is no .dynamic
			// section at all, which is exactly the outcome we want.
			if needed, derr := f.DynString(elf.DT_NEEDED); derr == nil && len(needed) > 0 {
				t.Errorf("binary still depends on shared libraries: %v", needed)
			}

			if goarch != runtime.GOARCH {
				return // cross-compiled, cannot execute it here
			}
			out, err := exec.Command(bin).CombinedOutput()
			if err != nil {
				t.Fatalf("run probe: %v\n%s", err, out)
			}
			got := string(out)
			if !strings.Contains(got, "available=false") {
				t.Errorf("ffi.Available() should report false in a static build, got:\n%s", got)
			}
			if !strings.Contains(got, "static_err=true") {
				t.Errorf("LoadLibrary should fail with ErrStaticBuild, got:\n%s", got)
			}
		})
	}
}

// writeProbeModule creates a throwaway module wired to this checkout, so the
// test needs no network access.
func writeProbeModule(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("locate module root: %v", err)
	}

	dir := t.TempDir()
	files := map[string]string{
		"main.go": probeSource,
		"go.mod": "module goffiprobe\n\ngo 1.26.0\n\n" +
			"require github.com/go-webgpu/goffi v0.0.0\n\n" +
			"replace github.com/go-webgpu/goffi => " + root + "\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// goToolPath returns the path to the go command. It prefers the go binary in
// the active GOROOT (as reported by "go env GOROOT"), falling back to whatever
// "go" resolves to on PATH. runtime.GOROOT was deprecated in Go 1.24 because
// the build-time GOROOT is meaningless once a binary is moved, so the value is
// queried from the tool at run time instead. Shared with musl_link_test.go.
func goToolPath() string {
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return "go"
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "go"
	}
	tool := filepath.Join(root, "bin", "go")
	if _, statErr := os.Stat(tool); statErr != nil {
		return "go"
	}
	return tool
}

func build(t *testing.T, dir, goarch, out string) {
	t.Helper()

	goTool := goToolPath()

	cmd := exec.Command(goTool, "build", "-tags", "goffi_static", "-o", out, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH="+goarch,
		"GOFLAGS=-mod=mod",
	)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build linux/%s with -tags goffi_static: %v\n%s", goarch, err, outBytes)
	}
}
