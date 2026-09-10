// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !cgo && linux && !android

package fakecgo

import (
	"unsafe"

	"github.com/go-webgpu/goffi/internal/hostlibc"
)

//go:nosplit
func _cgo_sys_thread_start(ts *ThreadStart) {
	var attr pthread_attr_t
	var ign, oset sigset_t
	var p pthread_t
	var size size_t
	var err int

	//fprintf(stderr, "runtime/cgo: _cgo_sys_thread_start: fn=%p, g=%p\n", ts->fn, ts->g); // debug
	sigfillset(&ign)
	pthread_sigmask(SIG_SETMASK, &ign, &oset)

	pthread_attr_init(&attr)
	pthread_attr_getstacksize(&attr, &size)
	// Leave stacklo=0 and set stackhi=size; mstart will do the rest.
	ts.g.stackhi = uintptr(size)

	err = _cgo_try_pthread_create(&p, &attr, unsafe.Pointer(threadentry_trampolineABI0), ts)

	pthread_sigmask(SIG_SETMASK, &oset, nil)

	if err != 0 {
		print("fakecgo: pthread_create failed: ")
		println(err)
		abort()
	}
}

// threadentry_trampolineABI0 maps the C ABI to Go ABI then calls the Go function
//
//go:linkname x_threadentry_trampoline threadentry_trampoline
var x_threadentry_trampoline byte
var threadentry_trampolineABI0 = &x_threadentry_trampoline

//go:nosplit
func threadentry(v unsafe.Pointer) unsafe.Pointer {
	ts := *(*ThreadStart)(v)
	free(v)

	setg_trampoline(setg_func, uintptr(unsafe.Pointer(ts.g)))

	// faking funcs in go is a bit a... involved - but the following works :)
	fn := uintptr(unsafe.Pointer(&ts.fn))
	(*(*func())(unsafe.Pointer(&fn)))()

	return nil
}

// here we will store a pointer to the provided setg func
var setg_func uintptr

//go:nosplit
func x_cgo_init(g *G, setg uintptr) {
	// Portable universal build. Two shims run before we touch any libc
	// symbol (the malloc below is the first). Both are no-ops in the default
	// and goffi_musl builds.
	//
	// 1. setupUniversalTLS: on the first, kernel-direct launch the thread
	//    pointer is unset (rt0_go delegates TLS setup to _cgo_init); give it a
	//    scratch page so the compiler's g-reloads after ABI0 calls don't fault.
	// 2. maybeReexecUniversal: re-exec through the host loader with libc
	//    pre-loaded, so the empty-SONAME imports bind. See
	//    reexec_universal_linux.go.
	setupUniversalTLS()
	maybeReexecUniversal()
	if hostlibc.Missing {
		// The bridge could not reach a libc, so every symbol this function
		// would use next -- malloc, the pthread_attr_* trio -- is unbound and
		// calling one faults. Give the process back to the Go runtime as a
		// pure-Go program instead of dying here.
		//
		// Clearing iscgo is what makes that work. The runtime read _cgo_init
		// long before this call and took the branch that delegates TLS setup
		// to us (setupUniversalTLS did it, onto a scratch page), but every
		// later decision -- creating an M with pthread_create versus clone(2),
		// keeping g in TLS versus the g register, installing signal handlers
		// the cgo way -- is made by reading runtime.iscgo at the point of use.
		// False from here on means the runtime never calls into the libc that
		// is not there, and threads it starts set up their own TLS.
		//
		// g.stacklo keeps the bounds rt0_go computed (SP minus 64 KiB); the
		// pthread_attr_getstacksize refinement below is exactly what a
		// CGO_ENABLED=0 binary does without, so nothing is lost.
		_iscgo = false
		setg_func = setg
		return
	}

	var size size_t
	var attr *pthread_attr_t

	/* The memory sanitizer distributed with versions of clang
	   before 3.8 has a bug: if you call mmap before malloc, mmap
	   may return an address that is later overwritten by the msan
	   library.  Avoid this problem by forcing a call to malloc
	   here, before we ever call malloc.

	   This is only required for the memory sanitizer, so it's
	   unfortunate that we always run it.  It should be possible
	   to remove this when we no longer care about versions of
	   clang before 3.8.  The test for this is
	   misc/cgo/testsanitizers.

	   GCC works hard to eliminate a seemingly unnecessary call to
	   malloc, so we actually use the memory we allocate.  */

	setg_func = setg
	attr = (*pthread_attr_t)(malloc(unsafe.Sizeof(*attr)))
	if attr == nil {
		println("fakecgo: malloc failed")
		abort()
	}
	pthread_attr_init(attr)
	pthread_attr_getstacksize(attr, &size)
	// runtime/cgo uses __builtin_frame_address(0) instead of `uintptr(unsafe.Pointer(&size))`
	// but this should be OK since we are taking the address of the first variable in this function.
	g.stacklo = uintptr(unsafe.Pointer(&size)) - uintptr(size) + 4096
	pthread_attr_destroy(attr)
	free(unsafe.Pointer(attr))
}
