// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && goffi_universal && (amd64 || arm64)

package fakecgo

import (
	"testing"

	"github.com/go-webgpu/goffi/internal/loader"
)

// The re-exec bridge (this package) and the public loader table must agree on
// the loader paths and libc SONAMEs, since both claim to describe the same host.
func TestReexecTableMatchesLoaderPackage(t *testing.T) {
	cases := []struct {
		name         string
		gotL, gotC   string
		wantL, wantC string
	}{
		{"glibc", glibcLoader, glibcLibc, loader.Glibc.Loader, loader.Glibc.LibC},
		{"musl", muslLoader, muslLibc, loader.Musl.Loader, loader.Musl.LibC},
	}
	for _, c := range cases {
		if c.gotL != c.wantL {
			t.Errorf("%s loader: bridge %q != loader pkg %q", c.name, c.gotL, c.wantL)
		}
		if c.gotC != c.wantC {
			t.Errorf("%s libc: bridge %q != loader pkg %q", c.name, c.gotC, c.wantC)
		}
	}
}
