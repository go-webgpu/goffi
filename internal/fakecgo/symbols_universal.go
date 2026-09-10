// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors
// SPDX-FileCopyrightText: 2026 Andrey Kolkov and GoGPU Contributors

//go:build !cgo && linux && !android && goffi_universal

// fakecgo dynamic imports for the portable "universal" Linux build.
//
// This mirrors symbols_linux.go but imports every libc entry point with an
// EMPTY SONAME, so the binary carries no DT_NEEDED and is not tied to glibc or
// musl. The symbols bind from the host libc after the re-exec bridge in
// reexec_universal_linux.go runs (before the Go runtime touches any of them).
//
// pthread_get_stacksize_np is intentionally absent: it is a Darwin-only API
// that neither glibc nor musl exports. glibc binds lazily, so the default
// build gets away with importing a stub nobody calls on Linux; musl binds
// eagerly. A universal binary must be loadable under either libc, so it must
// not name a symbol that musl cannot resolve. The stub's only caller is the
// Darwin thread-entry path, dead-code-eliminated on Linux.

package fakecgo

//go:cgo_import_dynamic goffi_malloc malloc ""
//go:cgo_import_dynamic goffi_free free ""
//go:cgo_import_dynamic goffi_setenv setenv ""
//go:cgo_import_dynamic goffi_unsetenv unsetenv ""
//go:cgo_import_dynamic goffi_sigfillset sigfillset ""
//go:cgo_import_dynamic goffi_nanosleep nanosleep ""
//go:cgo_import_dynamic goffi_abort abort ""
//go:cgo_import_dynamic goffi_sigaltstack sigaltstack ""
//go:cgo_import_dynamic goffi_pthread_attr_init pthread_attr_init ""
//go:cgo_import_dynamic goffi_pthread_create pthread_create ""
//go:cgo_import_dynamic goffi_pthread_detach pthread_detach ""
//go:cgo_import_dynamic goffi_pthread_sigmask pthread_sigmask ""
//go:cgo_import_dynamic goffi_pthread_self pthread_self ""
//go:cgo_import_dynamic goffi_pthread_attr_getstacksize pthread_attr_getstacksize ""
//go:cgo_import_dynamic goffi_pthread_attr_setstacksize pthread_attr_setstacksize ""
//go:cgo_import_dynamic goffi_pthread_attr_destroy pthread_attr_destroy ""
//go:cgo_import_dynamic goffi_pthread_mutex_lock pthread_mutex_lock ""
//go:cgo_import_dynamic goffi_pthread_mutex_unlock pthread_mutex_unlock ""
//go:cgo_import_dynamic goffi_pthread_cond_broadcast pthread_cond_broadcast ""
//go:cgo_import_dynamic goffi_pthread_setspecific pthread_setspecific ""

// No DT_NEEDED force-line, by design (see internal/dl/dl_universal.go).
