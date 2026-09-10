# musl / Alpine Builds (`-tags goffi_musl`)

## The problem

A default goffi binary does not start on Alpine, and it fails twice before
`main` ever runs:

1. **The interpreter is wrong.** The Go linker writes
   `PT_INTERP = /lib64/ld-linux-x86-64.so.2` — the glibc loader path. Alpine
   has no such file, so `execve` fails with a `no such file or directory`
   that misleadingly appears to be about the binary itself.

2. **The SONAMEs are wrong.** goffi's `//go:cgo_import_dynamic` directives
   name `libdl.so.2`, `libc.so.6` and `libpthread.so.0`. musl ships none of
   them: its entire POSIX surface — dlopen, pthreads, libm, errno — lives in
   one arch-named object, `libc.musl-x86_64.so.1` (or `-aarch64`). The musl
   dynamic linker refuses to start a process whose `DT_NEEDED` it cannot
   satisfy.

Both are baked into the ELF at link time, so no runtime cleverness can fix a
binary built for the wrong libc. The flavor is a build-time choice.

## Usage

```bash
CGO_ENABLED=0 go build -tags goffi_musl \
    -gcflags=github.com/go-webgpu/goffi/internal/dl=-std ./...
```

The same command works for `GOARCH=amd64` and `GOARCH=arm64`; the
architecture-specific loader path and SONAME are selected by build
constraints inside goffi.

The `-gcflags` part deserves a word. The musl interpreter path is baked in
with a `//go:cgo_dynamic_linker` directive, which the compiler restricts to
cgo-generated code; the flag relaxes that check for the one package that
carries it (`internal/dl`). Forgetting the flag is a loud compile error that
names the directive — deliberately preferable to the silent alternative, a
binary carrying the glibc interpreter that dies at startup on Alpine with a
confusing error. (This is the same mechanism some projects already use for
goffi's FreeBSD `fakecgo` shim.)

**Everything works in this mode.** Unlike `goffi_static`, which trades FFI
away for a static binary, `goffi_musl` is full-featured: `LoadLibrary`,
`GetSymbol`, `CallFunction`, callbacks, errno capture — all of it, resolved
from musl's libc. `ffi.Available()` reports `true`.

## What changes under the hood

| Package | glibc build | `goffi_musl` build |
|---|---|---|
| `internal/dl` | `dlopen` … from `libdl.so.2` | from `libc.musl-<arch>.so.1` |
| `internal/syscall` | `__errno_location` from `libc.so.6` | from `libc.musl-<arch>.so.1` |
| `internal/fakecgo` | `malloc`, `pthread_*` from `libc.so.6` / `libpthread.so.0` | from `libc.musl-<arch>.so.1` |
| ELF interpreter | `/lib64/ld-linux-x86-64.so.2` (linker default) | `/lib/ld-musl-<arch>.so.1` (directive) |

One symbol is intentionally absent from the musl set:
`pthread_get_stacksize_np` is a Darwin-only API that neither glibc nor musl
exports. The glibc build gets away with importing it because glibc binds
functions lazily and nobody calls the stub on Linux; musl binds every import
immediately at load time and would abort startup. Its trampoline is only
reachable from the Darwin thread-entry path, so on Linux the linker
dead-code-eliminates it. `TestMuslDirectiveParity` pins this exact
asymmetry, and keeps the glibc and musl symbol sets from drifting apart in
general.

The `internal/fakecgo` musl files are generated: `gen.go` produces them from
the same symbol tables as the glibc ones, filtered as described above.

## Tag interplay

- `goffi_musl` is meaningful only on Linux; elsewhere it selects nothing.
- `goffi_static` wins over `goffi_musl`: with both tags set, every dynamic
  import is compiled out and FFI is disabled, exactly as in a plain
  `goffi_static` build. A static binary is already libc-agnostic, so there
  is no musl flavor of it to want.

## Verification

`scripts/check-musl.sh` compiles both architectures, asserts the interpreter
and `DT_NEEDED` with `debug/elf` (`TestMuslLinkArtifacts`), and then executes
`cmd/musl-probe` inside a real Alpine userland — via `docker run alpine` on
CI, via a checksummed Alpine minirootfs and `chroot` when running as root
without docker, or via the musl loader invoked directly as a last resort.
The probe exercises every directive group the tag replaces: dlopen/dlsym,
integer and floating-point calls, errno capture through
`__errno_location`, C-to-Go callbacks (`qsort` with a Go comparator), and a
64-goroutine hammer that forces the Go runtime to create OS threads through
fakecgo's pthread imports.

## Choosing a Linux flavor

| You are shipping to | Build |
|---|---|
| glibc distros (Debian, Fedora, …) | default (no tags) |
| Alpine, postmarketOS, other musl distros | `-tags goffi_musl` + the `-gcflags` line above |
| `scratch` / distroless containers, no libc at all | `-tags goffi_static` (FFI off — there are no `.so` files to load there anyway) |

## Single binary for both libcs

Instead of separate glibc and `-tags goffi_musl` binaries, the universal build
produces one CGO-free binary that runs on both. See [PROFILE_U.md](PROFILE_U.md).
