// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi

import (
	"os"
	"strconv"
	"testing"
)

func TestExecutableUsesRecordedPath(t *testing.T) {
	t.Setenv(envUniversalExe, strconv.Itoa(os.Getpid())+":/opt/app/bin/app")
	got, err := Executable()
	if err != nil {
		t.Fatalf("Executable() error: %v", err)
	}
	if want := "/opt/app/bin/app"; got != want {
		t.Errorf("Executable() = %q, want %q", got, want)
	}
}

func TestExecutableIgnoresInheritedRecord(t *testing.T) {
	// A value tagged with another pid was inherited from a parent and
	// describes that parent's binary, not ours.
	t.Setenv(envUniversalExe, strconv.Itoa(os.Getpid()+1)+":/opt/parent/bin/parent")
	want, err := os.Executable()
	if err != nil {
		t.Skipf("os.Executable unavailable here: %v", err)
	}
	got, err := Executable()
	if err != nil {
		t.Fatalf("Executable() error: %v", err)
	}
	if got != want {
		t.Errorf("Executable() = %q, want os.Executable() %q", got, want)
	}
}

func TestArgv0(t *testing.T) {
	t.Setenv(envUniversalArgv0, strconv.Itoa(os.Getpid())+":f4")
	if got, want := Argv0(), "f4"; got != want {
		t.Errorf("Argv0() = %q, want %q", got, want)
	}

	t.Setenv(envUniversalArgv0, "")
	if got, want := Argv0(), os.Args[0]; got != want {
		t.Errorf("Argv0() without a record = %q, want %q", got, want)
	}
}

func TestRecordedSelfRejectsMalformed(t *testing.T) {
	for _, raw := range []string{
		"",                                    // unset
		"/opt/app/bin/app",                    // no pid tag
		"notapid:/opt/app/bin/app",            // unparsable tag
		strconv.Itoa(os.Getpid()) + ":",       // tagged, no value
		strconv.Itoa(os.Getpid()+1) + ":/opt", // another process
	} {
		t.Setenv(envUniversalExe, raw)
		if v, ok := recordedSelf(envUniversalExe); ok {
			t.Errorf("recordedSelf(%q) = %q, true; want false", raw, v)
		}
	}
}
