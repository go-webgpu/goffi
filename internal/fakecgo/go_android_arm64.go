// SPDX-License-Identifier: BSD-3-Clause
// SPDX-FileCopyrightText: 2011 The Go Authors
// SPDX-FileCopyrightText: 2026 The Goffi Authors

//go:build !cgo && android && arm64

package fakecgo

import "unsafe"

// Android reserves TLS_SLOT_APP (slot 2) for the runtime's g pointer on
// API-29+ arm64. Keep this as a package constant so the ABI test and startup
// guard cannot silently drift apart.
const androidTLSGOffset = uintptr(2 * unsafe.Sizeof(uintptr(0)))

// _cgo_sys_thread_start is the Android arm64 half of the fakecgo thread
// startup path. It intentionally mirrors runtime/cgo/gcc_linux_arm64.c while
// using Bionic's libc-only pthread symbols and Android-verified layouts.
//
//go:nosplit
func _cgo_sys_thread_start(ts *ThreadStart) {
	var attr pthread_attr_t
	var ign, oset sigset_t
	var p pthread_t
	var size size_t
	var err int

	sigfillset(&ign)
	pthread_sigmask(SIG_SETMASK, &ign, &oset)

	if pthread_attr_init(&attr) != 0 {
		androidFatal("fakecgo: pthread_attr_init failed on Android")
	}
	if pthread_attr_getstacksize(&attr, &size) != 0 {
		androidFatal("fakecgo: pthread_attr_getstacksize failed on Android")
	}
	// Leave stacklo=0 and set stackhi=size; mstart will do the rest.
	ts.g.stackhi = uintptr(size)

	err = _cgo_try_pthread_create(&p, &attr, unsafe.Pointer(threadentry_trampolineABI0), ts)

	pthread_sigmask(SIG_SETMASK, &oset, nil)

	if err != 0 {
		androidFatal("fakecgo: pthread_create failed on Android")
	}
}

// threadentry_trampolineABI0 maps the C ABI to Go ABI and then calls the Go
// function. The trampoline is supplied by the shared fakecgo arm64 assembly.
//
//go:linkname x_threadentry_trampoline threadentry_trampoline
var x_threadentry_trampoline byte
var threadentry_trampolineABI0 = &x_threadentry_trampoline

//go:nosplit
func threadentry(v unsafe.Pointer) unsafe.Pointer {
	ts := *(*ThreadStart)(v)
	free(v)

	setg_trampoline(setg_func, uintptr(unsafe.Pointer(ts.g)))

	fn := uintptr(unsafe.Pointer(&ts.fn))
	(*(*func())(unsafe.Pointer(&fn)))()

	return nil
}

// setg_func stores the runtime-provided setg_gcc callback for new pthreads.
var setg_func uintptr

// androidFatal is deliberately direct and nosplit: x_cgo_init may call it
// before normal runtime initialization, when panic, Go printing, and ordinary
// cgocall are not safe. It writes directly through Bionic and then aborts.
var androidFatalNewline = [1]byte{'\n'}

//go:nosplit
func androidFatal(message string) {
	if len(message) != 0 {
		write(2, unsafe.Pointer(unsafe.StringData(message)), size_t(len(message)))
	}
	write(2, unsafe.Pointer(&androidFatalNewline[0]), 1)
	abort()
}

// x_cgo_inittls validates the API-29+ Android arm64 TLS contract used by the
// pinned Go runtime. Android Q reserves TLS_SLOT_APP (slot 2), so runtime.tls_g
// must contain 2*sizeof(void*) == 16. The API-level guard runs first and uses
// direct dlsym/call5 calls; this function never enters runtime.cgocall.
//
//go:nosplit
func x_cgo_inittls(tlsg *uintptr, tlsbase unsafe.Pointer) {
	_ = tlsbase // The runtime supplies the base for the C implementation; slot 2 is fixed.
	if !androidAPI29() {
		androidFatal("fakecgo: Android API 29 or newer is required")
	}
	if tlsg == nil {
		androidFatal("fakecgo: Android runtime did not provide runtime.tls_g")
	}
	if *tlsg != androidTLSGOffset {
		androidFatal("fakecgo: Android runtime.tls_g offset mismatch (want 16)")
	}
}

// x_cgo_init matches the four-argument Android arm64 entry point in the
// pinned Go runtime: (G*, setg_gcc, &runtime.tls_g, TLS base).
//
//go:nosplit
func x_cgo_init(g *G, setg uintptr, tlsg *uintptr, tlsbase unsafe.Pointer) {
	setg_func = setg

	// The API/TLS guard must execute before touching the slot or entering any
	// ordinary cgocall path.
	x_cgo_inittls(tlsg, tlsbase)

	var size size_t
	var attr *pthread_attr_t

	attr = (*pthread_attr_t)(malloc(unsafe.Sizeof(*attr)))
	if attr == nil {
		androidFatal("fakecgo: malloc failed while initializing Android cgo")
	}
	if pthread_attr_init(attr) != 0 || pthread_attr_getstacksize(attr, &size) != 0 {
		androidFatal("fakecgo: pthread stack initialization failed on Android")
	}
	// runtime/cgo uses _cgo_set_stacklo with a malloc-backed probe. The
	// fakecgo path keeps the established Linux calculation, but applies it only
	// after the Bionic attr layout has been validated above.
	g.stacklo = uintptr(unsafe.Pointer(&size)) - uintptr(size) + 4096
	pthread_attr_destroy(attr)
	free(unsafe.Pointer(attr))
}
