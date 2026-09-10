# Profile U — one CGO-free binary for both glibc and musl

The universal ("Profile U") build produces a **single** `CGO_ENABLED=0` binary
that does live, in-process FFI on **both** glibc and musl systems (Debian,
Ubuntu, Alpine, …) — no C toolchain, no per-distro rebuild.

## Build

```sh
bash scripts/build-universal.sh -o app ./path/to/your/main
go run ./cmd/goffi-audit app      # asserts: no PT_INTERP, no DT_NEEDED
```

`scripts/build-universal.sh` builds with `-tags goffi_universal CGO_ENABLED=0`
and then strips the program interpreter. The universal build needs **no**
`-gcflags` (unlike the `goffi_musl` build).

## How it works

1. **No `DT_NEEDED`.** Every libc symbol is imported with an empty SONAME, so
   the linker records undefined symbols but pins the binary to no specific libc.
2. **No `PT_INTERP`.** The interpreter is stripped after linking, so the kernel
   loads the binary directly on any distribution (as it does a static binary).
3. **Re-exec through the host loader.** At the very top of startup — before any
   libc symbol is touched — the process re-execs itself through the host's own
   dynamic loader with the host libc pre-loaded. musl's loader binds the
   empty-SONAME symbols of a no-interp binary directly; glibc's loader only
   binds a main object that carries a `PT_INTERP`, so on glibc the binary hands
   the loader an in-memory copy of itself with the interpreter header restored.
   Either way the symbols bind against whichever libc the host ships.

The host loader and libc are discovered from an auditable table
(`internal/loader`), also exposed publicly:

```go
ffi.HostLoader() // e.g. "/lib64/ld-linux-x86-64.so.2" or "/lib/ld-musl-x86_64.so.1"
ffi.HostLibC()   // e.g. "libc.so.6" or "libc.musl-x86_64.so.1"
ffi.LibcKind()   // "glibc" | "musl" | "unknown"
```

## Limitations

- **Linux only.** amd64 is run-tested; arm64 is cross-compile-verified.
- A host whose dynamic loader goffi does not recognise cannot do FFI: there is
  nothing to re-exec through, so the process never binds a libc. It still
  runs. The bridge says so once on stderr, clears `runtime.iscgo` so the Go
  runtime creates threads with `clone(2)` and manages their TLS itself, and
  from then on the binary behaves like one built without FFI: `ffi.Available()`
  reports false, and `LoadLibrary`, `GetSymbol` and `CallFunction` return
  `ffi.ErrNoHostLibc` rather than jumping to an unbound symbol. Branch on
  `ffi.Available()` at startup if there is a pure-Go fallback to pick.
- After the re-exec, `argv[0]` is the image the loader was handed: the resolved
  executable path on musl, and on glibc the `/proc/self/fd/<n>` memfd copy with
  the interpreter header restored. `/proc/self/exe` — and therefore
  `os.Executable` — names the loader. Both are recorded before the re-exec and
  can be read back:

  ```go
  ffi.Executable() // where this binary lives on disk
  ffi.Argv0()      // what it was invoked as
  ```

  A program that re-runs, updates or installs itself, or that looks for files
  beside its own binary, wants those rather than `os` — see "Starting another
  copy of yourself" below.
- The universal build **owns the cgo runtime**; do not combine it with purego's
  fakecgo. To run goffi alongside purego, use the default build with
  `-tags nofakecgo` (see `docs/MUSL.md` and the CI `purego-coexistence` job).
  `-tags nofakecgo` is *not* an option in universal mode: dropping goffi's
  fakecgo also drops the re-exec bridge, which lives at the top of its
  `x_cgo_init`.

  If a dependency needs the purego *API*, prefer
  [`unxed/pureffi`](https://github.com/unxed/pureffi) (a drop-in replacement
  installed through a `replace` directive, already listed in the README): it
  implements that API entirely on top of goffi and carries **no fakecgo of its
  own**, so it needs no build tags and works unchanged in universal mode. No
  pureffi change is required for Profile U — this branch only *adds* public API
  (`HostLoader`/`HostLibC`/`LibcKind`) and build-tag-gated files.

## Starting another copy of yourself

`GOFFI_UNIVERSAL_REEXEC` is inherited, and the bridge honours it: a child that
inherits it concludes it already came through the loader and binds no libc.
Under `exec.Command(os.Args[0], ...)` — or any other plain respawn — the child
therefore dies before `main`, as `symbol lookup error: undefined symbol: malloc`
on glibc, or as a jump to an unbound symbol on musl.

Start the copy the way the bridge would have:

```go
exec.Command(ffi.HostLoader(), append(
    []string{"--preload", ffi.HostLibC(), os.Args[0]}, args...)...)
```

`os.Args[0]`, not `ffi.Executable()`: the loader has to be handed the image this
process was loaded from, which on glibc is the memfd and not the file on disk.
The loader shifts `argv` as usual, so the child sees the image as its `argv[0]`
and its own arguments from `argv[1]`.

## Attribution

The Profile U concept — an auditable "no `PT_INTERP`, no `DT_NEEDED`, reach the
host libc through its own loader" contract — is ported from
[`unxed/static-everywhere`](https://github.com/unxed/static-everywhere). The
in-process foreign-libc loader (`pg83/solo`) is intentionally **not** vendored;
goffi only needs the host's own libc, so it reuses the host loader via
`--preload`, exactly as static-everywhere endorses as the practical mechanism.
