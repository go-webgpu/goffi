// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && (darwin || freebsd || linux || netbsd) && !goffi_static

package fakecgo

import (
	"runtime"
	"sync"
	"testing"
)

// TestThreadEntryReturn exercises the fakecgo threadentry -> runtime.mstart
// return path, i.e. that an M created by fakecgo can exit cleanly.
//
// With iscgo forced true by fakecgo, the runtime creates every new OS thread
// through _cgo_thread_start -> threadentry_trampoline -> threadentry, which
// calls runtime.mstart. The M's stack is system-allocated, so when the M
// exits, mexit(osStack=true) returns from mstart back into threadentry with
// BP clobbered. On amd64 a frame-pointer LEAVE epilogue in threadentry (Go
// 1.28+) then faults on a NULL BP (ebitengine/purego#485).
//
// Locking an OS thread and letting the goroutine exit without
// UnlockOSThread forces the locked M to exit. A goroutine that never exits
// does not reproduce the bug, because mstart never returns.
func TestThreadEntryReturn(t *testing.T) {
	const rounds = 10
	const workers = 16
	for range rounds {
		var wg sync.WaitGroup
		wg.Add(workers)
		for range workers {
			go func() {
				defer wg.Done()
				runtime.LockOSThread()
				// Intentionally no UnlockOSThread: the goroutine exits with its
				// thread locked, so the M exits and mstart returns.
			}()
		}
		wg.Wait()
	}
}
