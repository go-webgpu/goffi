// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && (darwin || freebsd || linux || netbsd) && !android && !goffi_static

package fakecgo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSighowMatchesSystemHeaders checks SIG_BLOCK, SIG_UNBLOCK and SIG_SETMASK
// against <signal.h> of the system the test runs on.
//
// The values are not portable: Linux uses 0/1/2, Darwin and the BSDs 1/2/3.
// They were once defined in one shared file with the Linux values, so on
// Darwin and FreeBSD pthread_sigmask(SIG_SETMASK, ...) in
// _cgo_sys_thread_start ran as SIG_UNBLOCK: new runtime threads started with
// every signal unblocked instead of blocked (ebitengine/purego#552). Restating
// the numbers here would only repeat whatever the source says, so the test
// asks the platform's own C compiler instead and skips if there is none.
func TestSighowMatchesSystemHeaders(t *testing.T) {
	cc := findCC()
	if cc == "" {
		t.Skip("no C compiler found (set CC); cannot read <signal.h>")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "sighow.c")
	bin := filepath.Join(dir, "sighow")
	const prog = `#include <signal.h>
#include <stdio.h>
int main(void) { printf("%d %d %d\n", SIG_BLOCK, SIG_UNBLOCK, SIG_SETMASK); return 0; }
`
	if err := os.WriteFile(src, []byte(prog), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(cc, "-o", bin, src).CombinedOutput(); err != nil {
		t.Skipf("%s could not build the probe: %v\n%s", cc, err, out)
	}
	out, err := exec.Command(bin).Output()
	if err != nil {
		t.Fatalf("run probe: %v", err)
	}

	var block, unblock, setmask int
	if _, err := fmt.Sscan(strings.TrimSpace(string(out)), &block, &unblock, &setmask); err != nil {
		t.Fatalf("parse probe output %q: %v", out, err)
	}

	for _, c := range []struct {
		name string
		got  sighow
		want int
	}{
		{"SIG_BLOCK", SIG_BLOCK, block},
		{"SIG_UNBLOCK", SIG_UNBLOCK, unblock},
		{"SIG_SETMASK", SIG_SETMASK, setmask},
	} {
		if int(c.got) != c.want {
			t.Errorf("%s = %d, but <signal.h> on this system says %d", c.name, c.got, c.want)
		}
	}
}

// findCC returns $CC, or the first of cc, gcc and clang found on PATH.
func findCC() string {
	if cc := os.Getenv("CC"); cc != "" {
		return cc
	}
	for _, name := range []string{"cc", "gcc", "clang"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}
