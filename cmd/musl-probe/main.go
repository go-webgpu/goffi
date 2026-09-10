// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

// Command musl-probe is the runtime half of the goffi_musl verification.
//
// The link-time half (ffi/musl_link_test.go) proves the binary carries the
// right interpreter and SONAMEs; this program proves the machinery behind
// them actually works when executed against a real musl libc. Each check
// maps to one group of directives the goffi_musl tag replaces:
//
//	LoadLibrary/GetSymbol      -> internal/dl (dlopen/dlsym via libc.musl)
//	sqrt, strlen               -> the call path (float and integer returns;
//	                              on musl, libm lives inside libc)
//	getpid vs syscall.Getpid   -> a result checkable against ground truth
//	open() on a missing path   -> internal/syscall (__errno_location capture)
//	qsort with NewCallback     -> C-to-Go callbacks (crosscall2)
//	the goroutine hammer       -> internal/fakecgo (the runtime creates new
//	                              OS threads through _cgo_thread_start, i.e.
//	                              musl's pthread_create and friends)
//
// Exit status 0 and a final MUSL-PROBE-OK line mean every check passed. The
// program is built with -tags goffi_musl and run inside an Alpine userland
// by scripts/check-musl.sh and CI.
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

func muslLibc() string {
	switch runtime.GOARCH {
	case "amd64":
		return "libc.musl-x86_64.so.1"
	case "arm64":
		return "libc.musl-aarch64.so.1"
	default:
		return ""
	}
}

var failed bool

func check(name string, ok bool, detail string) {
	if ok {
		fmt.Printf("ok   %-22s %s\n", name, detail)
		return
	}
	failed = true
	fmt.Printf("FAIL %-22s %s\n", name, detail)
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
	lib := muslLibc()
	if lib == "" {
		fmt.Printf("FAIL unsupported GOARCH %s\n", runtime.GOARCH)
		os.Exit(1)
	}

	handle, err := ffi.LoadLibrary(lib)
	if err != nil {
		fmt.Printf("FAIL LoadLibrary(%s): %v\n", lib, err)
		os.Exit(1)
	}
	defer func() { _ = ffi.FreeLibrary(handle) }()
	check("LoadLibrary", true, lib)

	// sqrt(2.0): double(double). Exercises the SSE/FP register path.
	sqrtFn := mustSym(handle, "sqrt")
	sqrtCIF := mustCIF(types.DoubleTypeDescriptor, types.DoubleTypeDescriptor)
	arg := 2.0
	var root float64
	if _, err = ffi.CallFunction(sqrtCIF, sqrtFn,
		unsafe.Pointer(&root), []unsafe.Pointer{unsafe.Pointer(&arg)}); err != nil {
		fmt.Printf("FAIL CallFunction(sqrt): %v\n", err)
		os.Exit(1)
	}
	check("sqrt(2.0)", math.Abs(root-math.Sqrt2) < 1e-12, fmt.Sprintf("= %v", root))

	// strlen: size_t(char*). Integer return through RAX/X0.
	strlenFn := mustSym(handle, "strlen")
	strlenCIF := mustCIF(types.UInt64TypeDescriptor, types.PointerTypeDescriptor)
	s := "goffi on musl\x00"
	sp := unsafe.Pointer(unsafe.StringData(s))
	var n uint64
	if _, err = ffi.CallFunction(strlenCIF, strlenFn,
		unsafe.Pointer(&n), []unsafe.Pointer{unsafe.Pointer(&sp)}); err != nil {
		fmt.Printf("FAIL CallFunction(strlen): %v\n", err)
		os.Exit(1)
	}
	check("strlen", n == uint64(len(s)-1), fmt.Sprintf("= %d", n))

	// getpid: a value with independent ground truth on the Go side.
	getpidFn := mustSym(handle, "getpid")
	getpidCIF := mustCIF(types.SInt32TypeDescriptor)
	var pid int32
	if _, err = ffi.CallFunction(getpidCIF, getpidFn,
		unsafe.Pointer(&pid), nil); err != nil {
		fmt.Printf("FAIL CallFunction(getpid): %v\n", err)
		os.Exit(1)
	}
	check("getpid", int(pid) == syscall.Getpid(),
		fmt.Sprintf("C=%d Go=%d", pid, syscall.Getpid()))

	// open() on a path that cannot exist: return -1, errno ENOENT. This is
	// the __errno_location import doing real work on musl.
	openFn := mustSym(handle, "open")
	openCIF := mustCIF(types.SInt32TypeDescriptor,
		types.PointerTypeDescriptor, types.SInt32TypeDescriptor)
	path := "/goffi_musl_probe_nonexistent\x00"
	pathPtr := unsafe.Pointer(unsafe.StringData(path))
	flags := int32(0) // O_RDONLY
	var fd int32
	cerrno, err := ffi.CallFunction(openCIF, openFn,
		unsafe.Pointer(&fd),
		[]unsafe.Pointer{unsafe.Pointer(&pathPtr), unsafe.Pointer(&flags)})
	if err != nil {
		fmt.Printf("FAIL CallFunction(open): %v\n", err)
		os.Exit(1)
	}
	check("errno capture", fd == -1 && cerrno == syscall.ENOENT,
		fmt.Sprintf("ret=%d errno=%d", fd, cerrno))

	// qsort with a Go comparator: C calls back into Go through crosscall2.
	qsortFn := mustSym(handle, "qsort")
	qsortCIF := mustCIF(types.VoidTypeDescriptor,
		types.PointerTypeDescriptor, types.UInt64TypeDescriptor,
		types.UInt64TypeDescriptor, types.PointerTypeDescriptor)
	data := []int32{7, -3, 42, 0, -100, 13, 5, 5}
	cmp := ffi.NewCallback(func(a, b unsafe.Pointer) uintptr {
		va := *(*int32)(a)
		vb := *(*int32)(b)
		// Truncate to a C int in the low 32 bits; sign survives the trip.
		return uintptr(uint32(va - vb))
	})
	base := unsafe.Pointer(&data[0])
	nmemb := uint64(len(data))
	size := uint64(4)
	cmpArg := cmp
	if _, err := ffi.CallFunction(qsortCIF, qsortFn, nil, []unsafe.Pointer{
		unsafe.Pointer(&base), unsafe.Pointer(&nmemb),
		unsafe.Pointer(&size), unsafe.Pointer(&cmpArg),
	}); err != nil {
		fmt.Printf("FAIL CallFunction(qsort): %v\n", err)
		os.Exit(1)
	}
	check("qsort callback", sort.SliceIsSorted(data, func(i, j int) bool {
		return data[i] < data[j]
	}), fmt.Sprintf("%v", data))

	// Concurrency hammer: enough parallel FFI work that the Go runtime has
	// to create new OS threads, which under iscgo=true goes through
	// fakecgo's _cgo_thread_start -- pthread_create and the whole attr
	// family, now resolved from musl.
	runtime.GOMAXPROCS(max(4, runtime.NumCPU()))
	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for g := 0; g < 64; g++ {
		wg.Add(1)
		go func(seed float64) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				in := seed + float64(i)
				var out float64
				if _, err := ffi.CallFunction(sqrtCIF, sqrtFn,
					unsafe.Pointer(&out), []unsafe.Pointer{unsafe.Pointer(&in)}); err != nil {
					errs <- err
					return
				}
				if math.Abs(out*out-in) > 1e-6 {
					errs <- fmt.Errorf("sqrt(%v) = %v", in, out)
					return
				}
			}
		}(float64(g + 1))
	}
	wg.Wait()
	close(errs)
	hammerErr := <-errs
	check("thread hammer", hammerErr == nil, fmt.Sprintf("64 goroutines x 200 calls, err=%v", hammerErr))

	if failed {
		fmt.Println("MUSL-PROBE-FAILED")
		os.Exit(1)
	}
	fmt.Println("MUSL-PROBE-OK")
}
