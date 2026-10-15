# Platform Support

goffi's contract is a **zero-CGO** build: everything below is what you get with
`CGO_ENABLED=0`. `CGO_ENABLED=1` works too and shares the same ABI fast paths,
but no target in this table *requires* it.

`scripts/check-platforms.sh` owns this table. CI runs it on every push, so a row
here that disagrees with reality breaks the build.

## Tiers

| Tier      | `LoadLibrary` / `GetSymbol` | `CallFunction` | `NewCallback` |
| --------- | --------------------------- | -------------- | ------------- |
| `full`    | yes                         | yes            | yes           |
| `nocb`    | yes                         | yes            | fails explicitly |
| `load`    | yes                         | `ErrUnsupportedArchitecture` | yes (via `syscall.NewCallback`) |
| `pending` | not supported               | —              | —             |

## Matrix

| Platform | Arch    | Tier      | ABI              | Since  | CI                     |
| -------- | ------- | --------- | ---------------- | ------ | ---------------------- |
| Linux    | amd64   | `full`    | System V         | v0.1.0 | Runtime tested         |
| Linux    | arm64   | `full`    | AAPCS64          | v0.3.0 | Cross-compile verified |
| macOS    | amd64   | `full`    | System V         | v0.1.1 | Cross-compile verified |
| macOS    | arm64   | `full`    | AAPCS64          | v0.3.7 | Runtime tested (M-series) |
| Windows  | amd64   | `full`    | Win64            | v0.1.0 | Runtime tested         |
| Windows  | arm64   | `full`    | AAPCS64          | v0.5.0 | Cross-compile verified |
| FreeBSD  | amd64   | `full`    | System V         | v0.5.0 | Cross-compile verified |
| FreeBSD  | arm64   | `full`    | AAPCS64          | v0.5.3 | Cross-compile verified |
| NetBSD   | amd64   | `full`    | System V         | unreleased | Cross-compile verified |
| NetBSD   | arm64   | `full`    | AAPCS64          | unreleased | Cross-compile verified |
| Android  | arm64   | `nocb`    | AAPCS64 (Bionic) | v0.6.1 | Guarded preview (API 29+) |
| Windows  | 386     | `load`    | —                | —      | Cross-compile verified |
| Linux    | 386     | `pending` | cdecl            | —      | Tracked as pending     |
| Linux    | arm     | `pending` | AAPCS32          | —      | Tracked as pending     |
| Linux    | loong64 | `pending` | LP64D            | —      | Tracked as pending     |
| Linux    | ppc64le | `pending` | ELFv2            | —      | Tracked as pending     |
| Linux    | riscv64 | `pending` | LP64D            | —      | Tracked as pending     |
| Linux    | s390x   | `pending` | z/Architecture   | —      | Tracked as pending     |

### Build flags

FreeBSD and NetBSD need one extra flag under `CGO_ENABLED=0`:

```sh
CGO_ENABLED=0 go build \
  -gcflags="github.com/go-webgpu/goffi/internal/fakecgo=-std" ./...
```

`internal/fakecgo` publishes `environ` and `__progname` (plus `__ps_strings` on
NetBSD) through `//go:cgo_export_dynamic`, and that pragma is only accepted in
"std" packages. The symbols have to reach the **dynamic** symbol table, not just
the static one: on both systems `crt0` normally defines them and `libc.so`
carries undefined references that rtld resolves against the main object. A plain
`//go:linkname` produces a local definition only, and the process dies in rtld
before reaching `main`.

purego documents the same flag for these two platforms.

## Comparison with purego

purego is the reference for breadth, so the table below is the honest diff.
Rows marked *(cgo)* are `CGO_ENABLED=1`-only in purego and therefore out of
scope for goffi by design.

| Target            | purego | goffi     |
| ----------------- | ------ | --------- |
| linux/amd64       | Tier 1 | `full`    |
| linux/arm64       | Tier 1 | `full`    |
| darwin/amd64      | Tier 1 | `full`    |
| darwin/arm64      | Tier 1 | `full`    |
| windows/amd64     | Tier 1 | `full`    |
| windows/arm64     | Tier 1 | `full`    |
| freebsd/amd64     | Tier 2 | `full`    |
| freebsd/arm64     | Tier 2 | `full`    |
| netbsd/amd64      | Tier 2 | `full`    |
| netbsd/arm64      | Tier 2 | `full`    |
| windows/386       | Tier 2 (SyscallN + NewCallback only) | `load` (same shape) |
| android/arm64     | Tier 1 *(cgo)* | `nocb`, and without cgo |
| linux/386         | Tier 2 | `pending` |
| linux/arm         | Tier 2 | `pending` |
| linux/loong64     | Tier 2 | `pending` |
| linux/ppc64le     | Tier 2 | `pending` |
| linux/riscv64     | Tier 2 | `pending` |
| linux/s390x       | Tier 2 (needs cgo before Go 1.27) | `pending` |
| ios/amd64, ios/arm64 | Tier 1 *(cgo)* | out of scope |
| android/amd64, android/386, android/arm | *(cgo)* | out of scope |
| windows/arm       | Tier 2, dropped by Go 1.26 | out of scope |

goffi is ahead of purego in two places: android/arm64 runs without cgo, and
NetBSD exports `environ`/`__progname`/`__ps_strings` into the dynamic symbol
table so the binary can actually start under rtld.

## What "pending" costs

Each pending architecture needs roughly the same six pieces. This is why they
are tracked rather than stubbed: an FFI backend that compiles but gets register
assignment subtly wrong is worse than a clean `ErrUnsupportedArchitecture`.

1. `internal/syscall/syscall_<arch>.s` — the call trampoline that loads
   integer/float registers from the argument block, spills the rest to the
   outgoing stack area, calls the target, and writes the return registers back.
2. `internal/syscall/errno_stubs_<arch>.s` — in-trampoline errno capture, taken
   immediately after the C call returns while still on the same OS thread.
3. `internal/arch/<arch>/` — argument classification for that ABI plus the
   `Execute` that maps a `CallInterface` onto the register file.
4. `internal/fakecgo/asm_<arch>.s` + `trampolines_<arch>.s` + `abi_<arch>.h` —
   `crosscall2`, `threadentry`, `setg`, and the C↔Go ABI shims, without which
   `CGO_ENABLED=0` cannot start at all.
5. `internal/dl/dl_stubs_<arch>.s` + `dl_wrappers_<arch>.s` — the dlopen/dlsym
   bridge.
6. Callback trampolines, or an explicit `NewCallback` failure like android/arm64
   uses today.

Notes per architecture:

- **386** — cdecl passes everything on the stack, which makes the trampoline the
  simplest of the six, but 64-bit arguments must be split across two slots and
  float returns come back on the x87 stack (`ST(0)`), not in a register.
- **arm** — AAPCS32; armhf passes floats in VFP registers while Go's `linux/arm`
  has its own GOARM levels, so the soft-float/hard-float split has to be handled
  explicitly.
- **loong64**, **riscv64** — LP64D, closest in shape to AAPCS64; likely the
  cheapest two to land after the framework exists.
- **ppc64le** — ELFv2, with the TOC/r12 entry conventions and a separate
  stack-argument area.
- **s390x** — big-endian, so sub-8-byte integers are right-justified in their
  register slot while float32 is left-justified. Also needs Go 1.27 for a
  cgo-free build, matching purego.

See [ROADMAP.md](../ROADMAP.md) for sequencing.
