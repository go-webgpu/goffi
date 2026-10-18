// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build ((linux && !android) || darwin || freebsd) && !goffi_static

package ffi

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// dlerror() reports the last dynamic-linking error of the calling OS thread,
// and its string is valid only until the next dlerror() on that thread. A
// failed LoadLibrary or GetSymbol must read it on the thread whose dlopen or
// dlsym failed, before another goroutine can use that thread. Many goroutines
// fail with distinct names here; each error must name its own library or
// symbol. The text comes from the dynamic linker (errors.Unwrap skips goffi's
// LibraryError, which repeats the name itself).
func TestDlerror_ConcurrentFailures(t *testing.T) {
	libc, err := LoadLibrary(systemLibC(t))
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(4))

	const goroutines, iterations = 32, 300
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for g := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range iterations {
				var name string
				var err error
				if (g+i)%2 == 0 {
					name = fmt.Sprintf("/nonexistent/goffi-dlerror-%d-%d.so", g, i)
					_, err = LoadLibrary(name)
				} else {
					name = fmt.Sprintf("goffi_no_such_symbol_%d_%d", g, i)
					_, err = GetSymbol(libc, name)
				}
				if err == nil {
					errs <- fmt.Errorf("%s: no error", name)
					return
				}
				if msg := errors.Unwrap(err); msg == nil || !strings.Contains(msg.Error(), name) {
					errs <- fmt.Errorf("error for %s does not name it: %v", name, msg)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
