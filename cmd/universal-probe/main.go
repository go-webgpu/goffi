// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// Command universal-probe is the runtime half of the goffi_universal
// verification. One binary, built once with -tags goffi_universal and its
// PT_INTERP stripped, is expected to pass identically on a glibc host and on a
// musl host -- proving the re-exec-through-host-loader bridge really does bring
// up whichever libc is present.
//
// The checks mirror cmd/musl-probe: LoadLibrary (internal/dl), a float and an
// integer call (the call path), getpid against ground truth, a qsort callback
// (crosscall2), and a goroutine hammer that forces the runtime to spawn OS
// threads through fakecgo's pthread_create. The libc is loaded by whichever
// SONAME matches the host, discovered the same way the re-exec bridge does.
//
// Exit status 0 and a final UNIVERSAL-PROBE-OK line mean every check passed.
package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"sync"
	"syscall"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
)

// hostLibc returns the libc SONAME for the running host, chosen exactly like
// the re-exec bridge: prefer glibc if its loader is present, else musl.
func hostLibc() string {
	type pair struct{ loader, soname string }
	var glibc, musl pair
	switch runtime.GOARCH {
	case "amd64":
		glibc = pair{"/lib64/ld-linux-x86-64.so.2", "libc.so.6"}
		musl = pair{"/lib/ld-musl-x86_64.so.1", "libc.musl-x86_64.so.1"}
	case "arm64":
		glibc = pair{"/lib/ld-linux-aarch64.so.1", "libc.so.6"}
		musl = pair{"/lib/ld-musl-aarch64.so.1", "libc.musl-aarch64.so.1"}
	default:
		return ""
	}
	if _, err := os.Stat(glibc.loader); err == nil {
		return glibc.soname
	}
	if _, err := os.Stat(musl.loader); err == nil {
		return musl.soname
	}
	return ""
}

var failed bool

func check(name string, ok bool, detail string) {
	if ok {
		fmt.Printf("ok   %-20s %s\n", name, detail)
		return
	}
	failed = true
	fmt.Printf("FAIL %-20s %s\n", name, detail)
}

func mustSym(handle unsafe.Pointer, name string) unsafe.Pointer {
	sym, err := ffi.GetSymbol(handle, name)
	if err != nil {
		fmt.Printf("FAIL GetSymbol(%s): %v\n", name, err)
		os.Exit(1)
	}
	return sym
}

func mustCIF(ret *types.TypeDescriptor, args ...*types.TypeDescriptor) *types.CallInterface {
	cif := &types.CallInterface{}
	if err := ffi.PrepareCallInterface(cif, types.DefaultCall, ret, args); err != nil {
		fmt.Printf("FAIL PrepareCallInterface: %v\n", err)
		os.Exit(1)
	}
	return cif
}

func main() {
	reexeced := os.Getenv("GOFFI_UNIVERSAL_REEXEC") == "1"
	fmt.Printf("info re-exec bridge active: %v\n", reexeced)

	lib := hostLibc()
	if lib == "" {
		fmt.Printf("FAIL no known host libc for GOARCH %s\n", runtime.GOARCH)
		os.Exit(1)
	}

	handle, err := ffi.LoadLibrary(lib)
	if err != nil {
		fmt.Printf("FAIL LoadLibrary(%s): %v\n", lib, err)
		os.Exit(1)
	}
	defer func() { _ = ffi.FreeLibrary(handle) }()
	check("LoadLibrary", true, lib)

	// atof("2.0"): double(char*) -- FP return register path. atof lives in
	// libc on both glibc and musl (sqrt is in libm on glibc, so it would not
	// resolve from the libc handle on a glibc host).
	atofFn := mustSym(handle, "atof")
	atofCIF := mustCIF(types.DoubleTypeDescriptor, types.PointerTypeDescriptor)
	numStr := "2.0\x00"
	numPtr := unsafe.Pointer(unsafe.StringData(numStr))
	var val float64
	if _, err := ffi.CallFunction(atofCIF, atofFn,
		unsafe.Pointer(&val), []unsafe.Pointer{unsafe.Pointer(&numPtr)}); err != nil {
		fmt.Printf("FAIL CallFunction(atof): %v\n", err)
		os.Exit(1)
	}
	check("atof(\"2.0\")", math.Abs(val-2.0) < 1e-12, fmt.Sprintf("= %v", val))

	// strlen: size_t(char*) -- integer return.
	strlenFn := mustSym(handle, "strlen")
	strlenCIF := mustCIF(types.UInt64TypeDescriptor, types.PointerTypeDescriptor)
	s := "goffi universal\x00"
	sp := unsafe.Pointer(unsafe.StringData(s))
	var n uint64
	if _, err := ffi.CallFunction(strlenCIF, strlenFn,
		unsafe.Pointer(&n), []unsafe.Pointer{unsafe.Pointer(&sp)}); err != nil {
		fmt.Printf("FAIL CallFunction(strlen): %v\n", err)
		os.Exit(1)
	}
	check("strlen", n == uint64(len(s)-1), fmt.Sprintf("= %d", n))

	// getpid: checkable against the Go side.
	getpidFn := mustSym(handle, "getpid")
	getpidCIF := mustCIF(types.SInt32TypeDescriptor)
	var pid int32
	if _, err := ffi.CallFunction(getpidCIF, getpidFn, unsafe.Pointer(&pid), nil); err != nil {
		fmt.Printf("FAIL CallFunction(getpid): %v\n", err)
		os.Exit(1)
	}
	check("getpid", int(pid) == syscall.Getpid(), fmt.Sprintf("= %d", pid))

	// qsort with a Go comparator: C-to-Go callback via crosscall2.
	data := []int32{5, 3, 8, 1, 9, 2, 7, 4, 6, 0}
	cmp := ffi.NewCallback(func(a, b unsafe.Pointer) uintptr {
		x := *(*int32)(a)
		y := *(*int32)(b)
		switch {
		case x < y:
			return uintptr(^uint(0)) // -1
		case x > y:
			return 1
		default:
			return 0
		}
	})
	qsortFn := mustSym(handle, "qsort")
	qsortCIF := mustCIF(types.VoidTypeDescriptor,
		types.PointerTypeDescriptor, types.UInt64TypeDescriptor,
		types.UInt64TypeDescriptor, types.PointerTypeDescriptor)
	base := unsafe.Pointer(&data[0])
	count := uint64(len(data))
	size := uint64(unsafe.Sizeof(data[0]))
	if _, err := ffi.CallFunction(qsortCIF, qsortFn, nil, []unsafe.Pointer{
		unsafe.Pointer(&base), unsafe.Pointer(&count),
		unsafe.Pointer(&size), unsafe.Pointer(&cmp),
	}); err != nil {
		fmt.Printf("FAIL CallFunction(qsort): %v\n", err)
		os.Exit(1)
	}
	check("qsort callback", sort.SliceIsSorted(data, func(i, j int) bool { return data[i] < data[j] }),
		fmt.Sprintf("%v", data))

	// Goroutine hammer: force the runtime to create OS threads, which under a
	// cgo-enabled runtime go through fakecgo's pthread_create against the
	// host libc that the re-exec bridge pre-loaded.
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runtime.LockOSThread()
			var m uint64
			_, _ = ffi.CallFunction(strlenCIF, strlenFn,
				unsafe.Pointer(&m), []unsafe.Pointer{unsafe.Pointer(&sp)})
			runtime.UnlockOSThread()
		}()
	}
	wg.Wait()
	check("thread hammer", true, "32 goroutines locked to OS threads")

	if failed {
		fmt.Println("UNIVERSAL-PROBE-FAILED")
		os.Exit(1)
	}
	fmt.Println("UNIVERSAL-PROBE-OK")
}
