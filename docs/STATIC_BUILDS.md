# Static Builds (`-tags goffi_static`)

## The problem

A binary that imports goffi is dynamically linked, even with `CGO_ENABLED=0`
and even with `-ldflags "-extldflags '-static'"`:

```console
$ CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o app .
$ file app
app: ELF 64-bit LSB executable, x86-64, dynamically linked,
     interpreter /lib64/ld-linux-x86-64.so.2
$ ldd app
    libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6
    libdl.so.2 => /lib/x86_64-linux-gnu/libdl.so.2
```

Two things are going on:

1. **`//go:cgo_import_dynamic` is what makes the binary dynamic.** The Go
   linker writes a `PT_INTERP` header and `DT_NEEDED` entries as soon as any
   package in the build carries one of those directives. This happens under
   internal linking, with cgo disabled — the directives are the whole point of
   the mechanism.

2. **`-extldflags` is silently inert here.** It is passed to the *external*
   linker, and `CGO_ENABLED=0` builds never invoke one. The flag is accepted
   and ignored, which is why the comment in a build script can say "static"
   for years while the artifact is not.

goffi emits those directives from three places:

| Package | Symbols | Library |
|---------|---------|---------|
| `internal/dl` | `dlopen`, `dlsym`, `dlerror`, `dlclose` | `libdl.so.2` |
| `internal/syscall` | `__errno_location` | `libc.so.6` |
| `internal/fakecgo` | `malloc`, `free`, `pthread_*`, `sigaltstack`, … | `libc.so.6`, `libpthread.so.0` |

Removing any one of them is not enough; all three have to go.

## The trade

There is no way to keep FFI in a static binary on Linux. `dlopen` is a service
of the dynamic loader, and a static executable has no loader mapped into it —
glibc's static `dlopen` was always partial and was removed in 2.34, and musl
rejects it outright. Anything that resolves a symbol from a `.so` at runtime
needs `ld.so` in the process, and `ld.so` arrives only via `PT_INTERP`.

So the choice is per-build, not per-call:

|  | Dynamic (default) | `-tags goffi_static` |
|--|-------------------|----------------------|
| ELF interpreter | required | none |
| Runs on Alpine / `scratch` | no | yes |
| `LoadLibrary`, `GetSymbol` | work | return `ErrStaticBuild` |
| `CallFunction` | works | returns `ErrStaticBuild` |
| `NewCallback` | works | panics |
| `Available()` | `true` | `false` |

## Usage

```bash
CGO_ENABLED=0 go build -tags goffi_static ./...
```

The tag changes no API. Code compiles both ways, so a project can ship a static
artifact and a dynamic one from one source tree, and pick at build time which
backends it wants:

```go
if ffi.Available() {
    backend = newAcceleratedBackend() // GPU, Wayland, anything behind dlopen
} else {
    backend = newPureGoBackend()
}
```

`Available()` returns a compile-time constant, so the dead branch — and every
library binding reachable only from it — is eliminated by the linker rather
than shipped as unreachable code.

At the call site the error is ordinary and wrapped as usual:

```go
h, err := ffi.LoadLibrary("libvulkan.so.1")
if errors.Is(err, ffi.ErrStaticBuild) {
    // built without dynamic loading; fall back
}
```

## Where the tag does nothing

**Windows** resolves symbols through `LoadLibraryW`/`GetProcAddress`, not
through `cgo_import_dynamic`, and every Windows binary links `kernel32`
regardless. **Android** is dynamically linked by construction: Bionic loads the
app's `.so` files, and a static Android executable could not call into the
platform at all.

On both, the tag is accepted and ignored: `Available()` stays `true` and FFI
keeps working. A cross-platform build matrix can therefore pass
`-tags goffi_static` everywhere without special-casing mobile and Windows
targets, and only the platforms that can be static become static.
`scripts/check-static.sh` asserts this.

## Verifying

`scripts/check-static.sh` compiles the tagged build for every supported
platform and checks the linker output. The check that matters lives in
`ffi/static_link_test.go`: it builds a real consumer binary for linux/amd64 and
linux/arm64, opens it with `debug/elf`, and fails if a `PT_INTERP` header or
any `DT_NEEDED` entry survived. Inspecting the test binary itself would prove
nothing, since the test harness always links the dynamic build.

By hand:

```console
$ CGO_ENABLED=0 go build -tags goffi_static -o app .
$ file app
app: ELF 64-bit LSB executable, x86-64, statically linked
$ readelf -d app | head -1     # no .dynamic section at all
```

## Notes for future work

Two directions would narrow the trade-off above. Neither is implemented.

**Loader-agnostic symbol resolution.** The SONAMEs are currently hardcoded
(`libdl.so.2`, `libc.so.6`, `libpthread.so.0`), which is a glibc assumption: a
*dynamically* linked goffi binary still fails to start on Alpine, because musl
ships none of those names. Walking the process's own link map instead —
`PT_DYNAMIC` → `DT_DEBUG` → `r_debug.r_map` → each object's `.dynsym` — finds
`dlopen` in whatever libc is actually loaded, in pure Go, with no directives at
all. That would drop `internal/dl`'s directives, make dynamic builds work
unmodified on glibc, musl and Bionic, and leave the static/dynamic decision to
the linker rather than a build tag. `internal/fakecgo` would need the same
treatment (its assembly trampolines would jump through resolved pointers rather
than linker-supplied symbols) before the last `DT_NEEDED` entry disappears.

**A Go-native ELF loader.** Mapping a shared object with `mmap`, applying its
relocations and resolving its symbols — a minimal `ld.so` in Go — is the only
route to `dlopen` in a genuinely static binary. It is a large piece of work and
degrades quickly for libraries with deep dependency chains, but for a
self-contained `.so` it is tractable.
