// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

package ffi

import (
	"os"
	"strconv"
	"strings"
)

// Environment variables the universal ("Profile U") re-exec bridge writes for
// the process it re-execs; see internal/fakecgo/reexec_universal_linux.go.
// Each value is "<pid>:<value>", where pid is the process the bridge described.
// The tag matters because the environment is inherited: execve keeps the pid,
// so the re-executed process still matches, while a child gets a new pid and
// is told nothing about itself by a variable that describes its parent.
const (
	envUniversalExe   = "GOFFI_UNIVERSAL_EXE"
	envUniversalArgv0 = "GOFFI_UNIVERSAL_ARGV0"
)

// Executable returns the path of this program's executable file, the way
// os.Executable does, and stays right in a universal ("Profile U") build.
//
// Such a build has no PT_INTERP, so before main it re-execs itself through the
// host's dynamic loader with the host libc pre-loaded. After that execve
// /proc/self/exe -- what os.Executable reads on Linux -- names the loader, and
// on glibc, where the loader is handed a memfd copy of the binary, os.Args[0]
// names nothing that exists on disk. The path is therefore not recoverable
// from the running process; the bridge records it in the environment on the way
// through, and this returns what it recorded.
//
// Programs that re-run, update, or install themselves, or that look for files
// next to their own binary, want this rather than os.Executable. Everything
// else can keep calling os.Executable: outside a universal build the two are
// the same call.
func Executable() (string, error) {
	if p, ok := recordedSelf(envUniversalExe); ok {
		return p, nil
	}
	return os.Executable()
}

// Argv0 returns the name this process was invoked with -- what os.Args[0] would
// have held had the universal bridge not re-execed the process. It returns
// os.Args[0] unchanged outside a universal build, and "" only if os.Args is
// empty.
//
// This is the invocation name, not a path: it can be relative, or a bare name
// resolved through PATH, exactly as the caller wrote it. Use Executable to find
// the file.
func Argv0() string {
	if a, ok := recordedSelf(envUniversalArgv0); ok {
		return a
	}
	if len(os.Args) == 0 {
		return ""
	}
	return os.Args[0]
}

// recordedSelf reads a "<pid>:<value>" variable and reports its value if it
// describes this process. A non-empty value for another pid is a variable
// inherited from a parent, which says nothing about us.
func recordedSelf(key string) (string, bool) {
	raw := os.Getenv(key)
	if raw == "" {
		return "", false
	}
	tag, value, found := strings.Cut(raw, ":")
	if !found || value == "" {
		return "", false
	}
	pid, err := strconv.Atoi(tag)
	if err != nil || pid != os.Getpid() {
		return "", false
	}
	return value, true
}
