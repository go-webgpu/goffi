// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// Command universal-available prints whether the process has a libc to call
// into, and exits 0 either way. It is the test subject for hosts where the
// re-exec through the host loader cannot be trusted to work -- a library
// preloaded through /etc/ld.so.preload that aborts while it initialises -- and
// where a universal binary is expected to carry on without FFI instead of
// dying before main.
//
// Carrying on has to mean more than reaching main: a real program sets
// environment variables, starts goroutines on new threads and collects
// garbage. Each of those reaches a hook the cgo runtime installs, and a hook
// that calls into a libc that is not there is a jump to address 0. The first
// version of the fallback survived main and died on the first os.Setenv (f4
// #1213), so the program exercises them and prints "workload = ok" only when
// they all came back.
package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/go-webgpu/goffi/ffi"
)

func main() {
	fmt.Printf("ffi.Available = %v\n", ffi.Available())

	if err := os.Setenv("GOFFI_UNIVERSAL_TEST", "1"); err != nil || os.Getenv("GOFFI_UNIVERSAL_TEST") != "1" {
		fmt.Printf("workload = FAILED (Setenv: %v)\n", err)
		os.Exit(1)
	}
	if err := os.Unsetenv("GOFFI_UNIVERSAL_TEST"); err != nil || os.Getenv("GOFFI_UNIVERSAL_TEST") != "" {
		fmt.Printf("workload = FAILED (Unsetenv: %v)\n", err)
		os.Exit(1)
	}

	// New OS threads: with iscgo false the runtime must start them itself.
	var wg sync.WaitGroup
	total := make([]int, 8)
	for i := range total {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runtime.LockOSThread()
			for n := 0; n < 200000; n++ {
				total[i] += n & 1
			}
			runtime.Gosched()
		}()
	}
	wg.Wait()
	runtime.GC()

	fmt.Println("workload = ok")
}
