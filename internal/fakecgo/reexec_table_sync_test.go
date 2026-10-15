// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && !goffi_static && goffi_universal && (amd64 || arm64)

package fakecgo

import (
	"testing"

	"github.com/go-webgpu/goffi/internal/loader"
)

// The re-exec bridge (this package) and the public loader table must agree on
// the loader paths, libc SONAMEs and preload lists, since both claim to
// describe the same host -- and a program that starts another copy of itself
// through ffi.HostPreload has to hand the loader exactly what the bridge did.
func TestReexecTableMatchesLoaderPackage(t *testing.T) {
	cases := []struct {
		name                string
		gotL, gotC, gotP    string
		wantL, wantC, wantP string
	}{
		{"glibc", glibcLoader, glibcLibc, glibcPreload,
			loader.Glibc.Loader, loader.Glibc.LibC, loader.Glibc.Preload},
		{"musl", muslLoader, muslLibc, muslPreload,
			loader.Musl.Loader, loader.Musl.LibC, loader.Musl.Preload},
	}
	for _, c := range cases {
		if c.gotL != c.wantL {
			t.Errorf("%s loader: bridge %q != loader pkg %q", c.name, c.gotL, c.wantL)
		}
		if c.gotC != c.wantC {
			t.Errorf("%s libc: bridge %q != loader pkg %q", c.name, c.gotC, c.wantC)
		}
		if c.gotP != c.wantP {
			t.Errorf("%s preload: bridge %q != loader pkg %q", c.name, c.gotP, c.wantP)
		}
	}
}
