// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !cgo && linux && !android && !goffi_static

// Ported from Go's cmd/cgo/internal/test/issue1435.go via
// github.com/ebitengine/purego (syscall_linux_test.go, purego#426).

package fakecgo

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
)

// TestAllThreadsSyscall checks that fakecgo installed the cgo set*id hooks:
// with them, syscall.AllThreadsSyscall reports ENOTSUP exactly as in a real
// cgo binary. Without them it reaches runtime.doAllThreadsSyscall, which
// panics because iscgo is set.
func TestAllThreadsSyscall(t *testing.T) {
	_, _, err := syscall.AllThreadsSyscall(syscall.SYS_FCNTL, 0, 0, 0)
	if err != syscall.ENOTSUP {
		t.Errorf("AllThreadsSyscall should return ENOTSUP, got: %v", err)
	}
}

// TestSetuidEtc performs tests on all of the wrapped system calls that mirror
// the 9 glibc syscalls with POSIX semantics: a successful call must change the
// credentials of every thread of the process.
//
// As non-root every call must fail with EPERM (and must not panic). Run the
// test as root to check the per-thread credentials.
func TestSetuidEtc(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Log("not root: only checking that the calls fail with EPERM")
	}
	vs := []struct {
		call           string
		fn             func() error
		filter, expect string
	}{
		{call: "Setegid(1)", fn: func() error { return syscall.Setegid(1) }, filter: "Gid:", expect: "\t0\t1\t0\t1"},
		{call: "Setegid(0)", fn: func() error { return syscall.Setegid(0) }, filter: "Gid:", expect: "\t0\t0\t0\t0"},

		{call: "Seteuid(1)", fn: func() error { return syscall.Seteuid(1) }, filter: "Uid:", expect: "\t0\t1\t0\t1"},
		{call: "Setuid(0)", fn: func() error { return syscall.Setuid(0) }, filter: "Uid:", expect: "\t0\t0\t0\t0"},

		{call: "Setgid(1)", fn: func() error { return syscall.Setgid(1) }, filter: "Gid:", expect: "\t1\t1\t1\t1"},
		{call: "Setgid(0)", fn: func() error { return syscall.Setgid(0) }, filter: "Gid:", expect: "\t0\t0\t0\t0"},

		{call: "Setgroups([]int{0,1,2,3})", fn: func() error { return syscall.Setgroups([]int{0, 1, 2, 3}) }, filter: "Groups:", expect: "\t0 1 2 3"},
		{call: "Setgroups(nil)", fn: func() error { return syscall.Setgroups(nil) }, filter: "Groups:", expect: ""},
		{call: "Setgroups([]int{0})", fn: func() error { return syscall.Setgroups([]int{0}) }, filter: "Groups:", expect: "\t0"},

		{call: "Setregid(101,0)", fn: func() error { return syscall.Setregid(101, 0) }, filter: "Gid:", expect: "\t101\t0\t0\t0"},
		{call: "Setregid(0,102)", fn: func() error { return syscall.Setregid(0, 102) }, filter: "Gid:", expect: "\t0\t102\t102\t102"},
		{call: "Setregid(0,0)", fn: func() error { return syscall.Setregid(0, 0) }, filter: "Gid:", expect: "\t0\t0\t0\t0"},

		{call: "Setreuid(1,0)", fn: func() error { return syscall.Setreuid(1, 0) }, filter: "Uid:", expect: "\t1\t0\t0\t0"},
		{call: "Setreuid(0,2)", fn: func() error { return syscall.Setreuid(0, 2) }, filter: "Uid:", expect: "\t0\t2\t2\t2"},
		{call: "Setreuid(0,0)", fn: func() error { return syscall.Setreuid(0, 0) }, filter: "Uid:", expect: "\t0\t0\t0\t0"},

		{call: "Setresgid(101,0,102)", fn: func() error { return syscall.Setresgid(101, 0, 102) }, filter: "Gid:", expect: "\t101\t0\t102\t0"},
		{call: "Setresgid(0,102,101)", fn: func() error { return syscall.Setresgid(0, 102, 101) }, filter: "Gid:", expect: "\t0\t102\t101\t102"},
		{call: "Setresgid(0,0,0)", fn: func() error { return syscall.Setresgid(0, 0, 0) }, filter: "Gid:", expect: "\t0\t0\t0\t0"},

		{call: "Setresuid(1,0,2)", fn: func() error { return syscall.Setresuid(1, 0, 2) }, filter: "Uid:", expect: "\t1\t0\t2\t0"},
		{call: "Setresuid(0,2,1)", fn: func() error { return syscall.Setresuid(0, 2, 1) }, filter: "Uid:", expect: "\t0\t2\t1\t2"},
		{call: "Setresuid(0,0,0)", fn: func() error { return syscall.Setresuid(0, 0, 0) }, filter: "Uid:", expect: "\t0\t0\t0\t0"},
	}

	for i, v := range vs {
		// Generate some thread churn.
		c := make(chan struct{})
		go killAThread(c)
		close(c)

		if err := v.fn(); err != nil {
			if syscall.Getuid() == 0 && syscall.Getgid() == 0 {
				// If root, then the syscall should have succeeded.
				t.Errorf("[%d] %q setup failed: %v", i, v.call, err)
			} else if !errors.Is(err, syscall.EPERM) {
				// If not root, then EPERM is the only acceptable error.
				t.Errorf("[%d] %q unexpected error: %v", i, v.call, err)
			}
			continue
		}
		if err := compareStatus(v.filter, v.expect); err != nil {
			t.Errorf("[%d] %q comparison: %v", i, v.call, err)
		}
	}
}

// killAThread locks the goroutine to an OS thread and exits; this causes an
// OS thread to terminate.
func killAThread(c <-chan struct{}) {
	runtime.LockOSThread()
	<-c
}

// compareStatus checks that the filter line of every /proc/<pid>/task/<tid>
// status file of this process matches expect.
func compareStatus(filter, expect string) error {
	expected := filter + expect
	pid := syscall.Getpid()
	fs, err := os.ReadDir(fmt.Sprintf("/proc/%d/task", pid))
	if err != nil {
		return fmt.Errorf("unable to find %d tasks: %w", pid, err)
	}
	expectedProc := fmt.Sprintf("Pid:\t%d", pid)
	foundAThread := false
	for _, f := range fs {
		tf := fmt.Sprintf("/proc/%s/status", f.Name())
		d, err := os.ReadFile(tf)
		if err != nil {
			// There are a surprising number of ways this can error out on
			// linux (not exist, no such process, bad file descriptor), so
			// treat any error here as "the thread is gone".
			continue
		}
		for line := range strings.SplitSeq(string(d), "\n") {
			// Different kernel vintages pad differently.
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Pid:\t") {
				// On loaded systems a TID can be reused quickly, so check
				// that this thread still belongs to our process.
				if line != expectedProc {
					break
				}
			}
			if strings.HasPrefix(line, filter) {
				if line == expected {
					foundAThread = true
					break
				}
				if filter == "Groups:" && strings.HasPrefix(line, "Groups:\t") {
					// https://github.com/golang/go/issues/46145
					// Containers don't reliably output this line in sorted
					// order, so sort it before comparing.
					a := strings.Split(line[8:], " ")
					slices.Sort(a)
					if strings.Join(a, " ") == expected[8:] {
						foundAThread = true
						break
					}
				}
				return fmt.Errorf("%q got:%q want:%q (bad) [pid=%d file:'%s' %v]", tf, line, expected, pid, string(d), expectedProc)
			}
		}
	}
	if !foundAThread {
		return fmt.Errorf("found no thread /proc/<TID>/status files for process %q", expectedProc)
	}
	return nil
}
