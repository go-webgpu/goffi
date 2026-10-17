// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build linux || darwin || freebsd

// Command linktest checks that goffi links and works in one program together
// with other code that calls C (#72): ebitengine/purego, and an ordinary cgo
// package (cgo builds only). It calls getpid through goffi, purego and cgo,
// compares the results with os.Getpid, and checks that goffi captures errno
// (close(-1) must fail with EBADF).
//
// It is a separate module under testdata so that purego never becomes a
// dependency of goffi. scripts/run-linktest.sh runs it in every link mode.
package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/ebitengine/purego"

	"linktest/goffiprobe"
)

// cgoGetpid is set by cgo_on.go when the program is built with cgo.
var cgoGetpid func() int32

func main() {
	want := int32(os.Getpid())
	cgoMode := "cgo off"
	if cgoGetpid != nil {
		cgoMode = "cgo on"
	}
	fmt.Printf("linktest: %s/%s, %s, %s, os.Getpid=%d\n", runtime.GOOS, runtime.GOARCH, runtime.Version(), cgoMode, want)

	var failed []string
	check := func(name string, ok bool, detail string) {
		status := "ok"
		if !ok {
			status = "FAIL"
			failed = append(failed, name)
		}
		fmt.Printf("  %-22s %-4s %s\n", name, status, detail)
	}

	pid, err := goffiprobe.Getpid()
	check("goffi getpid", err == nil && pid == want, fmt.Sprintf("got %d, err %v", pid, err))

	ret, errno, err := goffiprobe.CloseInvalidErrno()
	check("goffi errno", err == nil && ret == -1 && errno == syscall.EBADF,
		fmt.Sprintf("close(-1) = %d, errno %d (%v), err %v; want -1, EBADF", ret, int(errno), errno, err))

	lib, err := purego.Dlopen(goffiprobe.LibC(), purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		check("purego getpid", false, fmt.Sprintf("Dlopen: %v", err))
	} else {
		var getpid func() int32
		purego.RegisterLibFunc(&getpid, lib, "getpid")
		got := getpid()
		check("purego getpid", got == want, fmt.Sprintf("got %d", got))
	}

	if cgoGetpid != nil {
		got := cgoGetpid()
		check("cgo getpid", got == want, fmt.Sprintf("got %d", got))
	}

	if len(failed) > 0 {
		fmt.Printf("linktest: FAIL %v\n", failed)
		os.Exit(1)
	}
	fmt.Println("linktest: PASS")
}
