# Android arm64 preview candidate

goffi carries an Android arm64/API 29+ preview candidate. Cross-build, ABI,
and ELF probes pass, but physical-device startup validation is still required
before this can be described as released Android support.

The implementation follows the pinned Go runtime's Android AAPCS64 startup
contract. The four-argument `_cgo_init` entry point and TLS setup are
irreducible: the runtime passes `(g, setg_gcc, &runtime.tls_g, TLS base)` before
ordinary `runtime.cgocall` is available, then reads the Go `g` pointer from
Bionic's `TLS_SLOT_APP` (slot 2). The fakecgo trampoline must preserve all four
registers, validate API/TLS first, and use Bionic's LP64 pthread and signal
layouts; a generic Linux startup path would corrupt runtime state before Go
could report an error.

Both build modes are candidate build surfaces:

```sh
# No C compiler is needed for this mode.
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build ./...

# For applications that already use cgo, point CC at the API-29 NDK driver.
CC="$ANDROID_NDK_HOME/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android29-clang" \
GOOS=android GOARCH=arm64 CGO_ENABLED=1 go build ./...
```

The cgo=0 path uses direct Bionic `libc.so`/`libdl.so` dynamic imports and the
fakecgo startup path. The cgo=1 path uses small NDK C wrappers so external
linking never emits an AAPCS64 branch relocation to a `cgo_import_dynamic`
symbol. Both paths reject glibc sonames and `__errno_location`.

`runtime.iscgo` is intentionally true in the cgo=0 path. Android builds also
satisfy Go's `linux` build term, so the shared `iscgo.go`, `callbacks.go`, and
`setenv.go` wiring is selected. `runtime.cgocall` rejects ordinary Unix targets
when `iscgo` is false; when it is true, the runtime also selects its cgo-aware
thread, TLS, signal, traceback, and extra-M paths. Android fakecgo supplies the
init, thread-start, environment, pthread-key, and bind hooks those paths expect.
This runtime wiring is separate from goffi's public callback policy.

Android callback trampolines are deliberately unavailable. `ffi.NewCallback`
panics with a stable message instead of exposing a pointer whose foreign-thread
startup path has not been validated on a physical device. Vulkan/WebGPU users
should use polling or an application-owned native callback bridge until that
evidence exists.

Dynamic-library handles are retained for process lifetime. `RTLD_NOW | RTLD_LOCAL
| RTLD_NODELETE` makes that policy explicit, and `FreeLibrary` is safe to call
but does not unload code that may still have function pointers in use.

## Regression probe

The NDK header, Go layout, cgo=0/cgo=1 cross-build, and ELF dependency checks
are reproducible without a device:

```sh
ANDROID_NDK_HOME=/path/to/android-ndk-r29 scripts/check-android-arm64.sh
```

The audited source/ABI matrix is Go 1.25.12 and Go 1.26.5 with Android NDK
r29 (`29.0.14206865`). Keep both Go lines in CI when runtime startup files or
TLS offsets change upstream. Passing this probe is not physical-device
startup evidence.
