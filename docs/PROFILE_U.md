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
   dynamic loader with the host libc pre-loaded (on glibc together with
   `libpthread.so.0` and `libdl.so.2`: before glibc 2.34 the `pthread_*` and
   `dl*` functions live there, not in `libc.so.6`). musl's loader binds the
   empty-SONAME symbols of a no-interp binary directly; glibc's loader only
   binds a main object that carries a `PT_INTERP`, so on glibc the binary hands
   the loader an in-memory copy of itself with the interpreter header restored.
   Either way the symbols bind against whichever libc the host ships.

The host loader and libc are discovered from an auditable table
(`internal/loader`), also exposed publicly:

```go
ffi.HostLoader() // e.g. "/lib64/ld-linux-x86-64.so.2" or "/lib/ld-musl-x86_64.so.1"
ffi.HostLibC()   // e.g. "libc.so.6" or "libc.musl-x86_64.so.1"
ffi.HostPreload() // e.g. "libc.so.6 libpthread.so.0 libdl.so.2" or "libc.musl-x86_64.so.1"
ffi.LibcKind()   // "glibc" | "musl" | "unknown"
```

## Limitations

- **Linux only.** amd64 is run-tested; arm64 is cross-compile-verified.
- **glibc 2.30 or newer.** The loader's `--preload` option first appeared in
  glibc 2.30. An older loader takes `--preload` for the name of the program to
  run and the re-executed process dies before `main` with
  `--preload: cannot open shared object file` (observed on Ubuntu 18.04,
  glibc 2.27). The oldest glibc CI runs the universal binary on is 2.31
  (Debian 11, Ubuntu 20.04).
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
  `-tags nofakecgo` (the CI `purego-coexistence` job builds that combination).
  `-tags nofakecgo` is *not* an option in universal mode: dropping goffi's
  fakecgo also drops the re-exec bridge, which lives at the top of its
  `x_cgo_init`.

## Hosts that preload a library

A library named in `/etc/ld.so.preload`, or in `LD_PRELOAD`, is initialised in
every process the host loader starts -- including the re-executed one. If its
constructor aborts there (ESET's `libesets_pac.so` does on some hosts, f4
#1213), the process dies before `main`: it is the re-executed process, not the
one that could still have chosen to go on without FFI.

So when either is non-empty, the bridge first starts the same launch -- same
loader, libc, image and environment -- in a forked child, with
`GOFFI_UNIVERSAL_PROBE=<pid>:1` added. That child stops as soon as it reaches the
bridge, so nothing of the program runs. If it exits 0 the real re-exec goes
ahead; if it was killed or failed, the bridge says so once on stderr and
continues without FFI, exactly as on a host with no known loader
(`ffi.Available()` is false). A probe that cannot be carried out at all (fork
refused, no scratch memory) is not held against the launch. With nothing
preloaded there is no probe and no cost.

The probe variable is set only in the child's environment; the running process
never sees it.

## Starting another copy of yourself

Just start it: `exec.Command(os.Args[0], ...)` or `exec.Command(exe, ...)`
with `exe` from `ffi.Executable()`. The child runs the bridge itself and gets a
working libc.

Starting the copy through the host loader by hand
(`exec.Command(ffi.HostLoader(), "--preload", ffi.HostPreload(), os.Args[0], ...)`)
is no longer needed. It still works on musl and on glibc 2.34+, but glibc 2.31's
loader rejects the `/proc/self/fd/<n>` image with `loader cannot load itself`
(Debian 11, Ubuntu 20.04), before any goffi code runs.

The guard the bridge leaves in the environment is `GOFFI_UNIVERSAL_REEXEC=<pid>:1`,
tagged like `GOFFI_UNIVERSAL_EXE` and `GOFFI_UNIVERSAL_ARGV0`. The re-executed
process keeps its pid across `execve` and finds its own tag. A child inherits
the variable but has a new pid, so it runs the bridge itself. The bridge also
drops the inherited copies of all four variables before writing its own, so
they do not pile up across generations.

Two ways of being started need no re-exec, and the bridge recognises both:

- by the host loader as a program (`/proc/self/exe` is the loader): the loader
  has already preloaded the libc it was given;
- from the parent's memfd (on glibc a re-executed process has `os.Args[0]` =
  `/proc/self/fd/<n>`, and a child started with it runs that image). This one
  does go through the bridge again, and records the parent's `ffi.Executable()`
  as its own, since `/proc/self/exe` names no file.

`ffi.Executable()` therefore names the file on disk in every child.
`cmd/universal-respawn` starts itself through `os.Args[0]`, `ffi.Executable()`,
the host loader and a grandchild, and checks that each child makes FFI calls
(the host-loader launch is reported but not required). CI runs it on glibc and
musl.

## Attribution

The Profile U concept — an auditable "no `PT_INTERP`, no `DT_NEEDED`, reach the
host libc through its own loader" contract — is ported from
[`unxed/static-everywhere`](https://github.com/unxed/static-everywhere). The
in-process foreign-libc loader (`pg83/solo`) is intentionally **not** vendored;
goffi only needs the host's own libc, so it reuses the host loader via
`--preload`, exactly as static-everywhere endorses as the practical mechanism.
