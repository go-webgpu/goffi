# Profile U — universal (glibc + musl) FFI: task, plan, and status

> This document is a self-contained hand-off. It is written so that a new
> contributor (or a smaller model) can continue the work without access to the
> original conversation. It states the goal, the full end-to-end plan, exactly
> what is already done, the one blocker currently in the way (with its designed
> fix), and the precise next steps with commands.
>
> **Status: the plan below is complete.** The universal path works end to end on
> both glibc and musl (§5.1/§5.1b), the loader package, audit tool, tests, CI and
> docs (§5.2-§5.7) have landed. Section 3 is kept as the historical record of the
> blocker that was fixed. See "START HERE" for what is left.

---

## 1. The task (what we are building and why)

goffi is a pure-Go, CGO-free FFI library (a fork of ebitengine/purego, module
path `github.com/go-webgpu/goffi`). On Linux it makes C calls by importing libc
symbols via `//go:cgo_import_dynamic`, which makes the Go linker emit a
`DT_NEEDED` for a specific libc SONAME plus a fixed `PT_INTERP`. That pins a
binary to one libc: the default build needs glibc (`libc.so.6`,
`/lib64/ld-linux-x86-64.so.2`), and the `goffi_musl` build is a *separate*
binary for musl (`libc.musl-<arch>.so.1`, `/lib/ld-musl-<arch>.so.1`).

**Goal:** port the "Profile U" (universal) idea from `unxed/static-everywhere`
into this fork so that a *single* binary does live, in-process FFI on **both**
glibc and musl systems, with **no C toolchain** (CGO_ENABLED=0 is the whole
point — the result must be fully portable). Concretely the deliverables are:

1. A `goffi_universal` build that yields one binary running FFI on glibc *and*
   musl.
2. Port "the loader" — the Profile U loader concept — in a form appropriate to
   goffi (see §4; we do **not** vendor a full in-process ELF loader).
3. Ensure goffi does **not conflict with purego** — specifically the
   CGO_ENABLED=0 fakecgo symbol collision (`duplicated definition of symbol
   _cgo_init`). See §7.
4. Tests **and CI that exercise both libc** on GitHub.
5. Deliver on a branch of the fork (`feature/profile-u-musl`).

Non-negotiable constraint from the user: **no cgo.** Center everything on the
portable CGO_ENABLED=0 path. The purego-coexistence guarantee that matters is
therefore the CGO_ENABLED=0 one.

### What "Profile U" means here (scope decision)

static-everywhere's Profile U is defined by an auditable ELF contract —
"ET_DYN/EXEC, **no PT_INTERP, no DT_NEEDED**; host libraries reached only
through a carried loader/ABI bridge" — and its own docs endorse, as the
*practical* mechanism, "launch with the host's own loader, found at runtime —
`/lib64/ld-linux-x86-64.so.2`, else `/lib/ld-musl-x86_64.so.1` … one binary
works on either libc." The full in-process foreign-libc loader (`pg83/solo`) is
explicitly declared out of scope there ("multi-year, bad safety story").

goffi's need is smaller than static-everywhere's: goffi only needs the *host's
own* libc (glibc on a glibc box, musl on a musl box), not to load glibc
libraries on a musl host. So we adopt exactly the endorsed mechanism and do
**not** vendor SoLo. This is faithful to the source and is what makes the
problem tractable.

---

## 2. The validated design (proven on real glibc and musl)

Key fact: glibc and musl export the **same libc symbol names** (`dlopen`,
`dlsym`, `__errno_location`, `malloc`, `pthread_*`, …). Only the SONAME strings
and the interpreter path differ. So one binary can bind against either libc if
it (a) names no specific SONAME and (b) is launched so that *some* libc is
mapped and its symbols are visible.

The mechanism, each part empirically validated in the dev sandbox:

1. **Empty-SONAME imports → no `DT_NEEDED`.**
   `//go:cgo_import_dynamic goffi_dlopen dlopen ""` (empty remote library) emits
   an *undefined* dynamic symbol with **no** `DT_NEEDED`. Omitting the
   `//go:cgo_import_dynamic _ _ "libc.so.6"` "force-line" is what removes the
   `DT_NEEDED`. Verified: the resulting binary names no libc SONAME.

2. **Strip `PT_INTERP`.** The Go linker still writes the default glibc
   interpreter even with empty-SONAME imports. A musl-only box has no
   `/lib64/ld-linux-x86-64.so.2`, so the kernel would refuse to exec the binary
   there and our code would never run. We therefore flip the `PT_INTERP`
   program header to `PT_NULL` after linking (`cmd/goffi-strip-interp`). With no
   interpreter the kernel loads the binary directly on every distro (as it does
   for fully static binaries), and the Go runtime starts fine (verified: a
   no-interp Go program that does not touch libc runs cleanly).

3. **Early re-exec through the host loader with libc pre-loaded.** A no-DT_NEEDED
   binary has unbound libc symbols; something must map a libc. We re-exec the
   process through the host's *own* dynamic loader with the host libc
   pre-loaded:

   ```
   execve(<host-loader>,
          {<host-loader>, "--preload", <host-libc-soname>, <self>, <argv[1:]...>},
          <envp>)
   ```

   The host loader maps its libc; the global symbol scope then contains
   `malloc`, `dlopen`, `pthread_*`, `__errno_location`; and the re-executed
   process binds them from whichever libc the host ships. **Verified on both
   libc:** `ld.so --preload libc.so.6 ./bin` (glibc) and
   `ld-musl-x86_64.so.1 --preload libc.musl-x86_64.so.1 ./bin` (musl) both bind
   the undefined no-DT_NEEDED symbols and a real libc call returns correctly.
   Bare SONAMEs work on both (each loader resolves its own libc via its default
   search path). The musl loader supports `--preload`.

   After the re-exec, goffi's existing FFI machinery + fakecgo works unchanged
   (independently verified earlier: the default glibc goffi binary ran the full
   FFI probe — including a thread hammer — under the musl loader).

This needs **no in-process ELF loader** (no SoLo): we reuse the host's own
`ld.so`, which is present on every distro. The "carried loader" is just a small
discovery table + the re-exec bridge.

### Where the re-exec must happen (timing)

`x_cgo_init` (fakecgo's cgo init) is called from `rt0_go`
(`runtime/asm_amd64.s:190`, `asm_arm64.s:124`) **before** `runtime·args`
(line 338) and **before** the scheduler/sysmon start. Under `iscgo=true`, new
OS threads are created via `_cgo_thread_start` → `pthread_create`, which is
unbound on the first (libc-less) launch. The first libc touch is the `malloc`
at the top of `x_cgo_init`. **Therefore the re-exec must run at the very top of
`x_cgo_init`, before `malloc`,** using only raw syscalls (no libc, no Go heap).
argc/argv are not yet in runtime globals there, so argv/envp are read from
`/proc/self/cmdline` and `/proc/self/environ` (both NUL-delimited) and
pointer-ified in place; scratch memory comes from raw `mmap`.

---

## 3. The startup blocker and its fix (RESOLVED - historical record)

### Symptom
A universal binary (empty-SONAME + stripped interp + re-exec at top of
`x_cgo_init`) **segfaults at startup on the first launch**, before reaching
`execve`. A single early `write(2)` diagnostic prints, then it dies.

### Root cause (confirmed)
When `_cgo_init` is present, `rt0_go` **skips** the runtime's own TLS setup and
delegates it to `_cgo_init`. In `runtime/asm_amd64.s`, the `JZ needtls` (jump to
the TLS-setup path) is taken **only when `_cgo_init == nil`**. With fakecgo,
`_cgo_init` is non-nil, so `%fs`/TLS is **not** set up when `x_cgo_init` runs.

The Go compiler emits `MOVQ FS:0xfffffff8, R14` (reload the `g` register from
TLS at `%fs-8`) **after every call to an ABI0 assembly function** (e.g. our
`rawsyscall6`, or fakecgo's `call5`). On a no-interpreter binary the kernel
loaded directly, `%fs` is unset (0), so that TLS read faults. This is why the
first `write` succeeds (the syscall runs) but the instruction right after it
(the g-reload) crashes. In the default/musl builds this never happens because
the host `ld.so` sets up `%fs` before the Go entry point runs.

Verified with `go tool objdump` on the built binary: the fault is exactly the
`MOVQ FS:-8, R14` after the first ABI0 call.

### The fix (implemented; landed in `setup_universal_tls_*.s`)
Add a tiny per-arch **assembly** shim `setupUniversalTLS()` and call it as the
**very first statement** of `x_cgo_init` in the universal build, *before*
`maybeReexecUniversal()`. Because it is the first `CALL` in `x_cgo_init` and it
sets `%fs` *before returning*, the compiler's post-call g-reload then reads
mapped memory. The fake `g` value is never dereferenced before `execve` (our
re-exec code uses only raw syscalls and mmap/rodata memory; write barriers are
gated on `runtime.writeBarrier.enabled`, which is false this early — and we
already store the staging-buffer base as `uintptr` to avoid a barrier anyway).
On the **re-executed** launch the host loader sets up real TLS, so the shim is a
no-op.

Implementation sketch:

- **amd64** (`setup_universal_tls_amd64.s`, build `... && goffi_universal && amd64`):
  ```
  SYS_arch_prctl = 158 ; ARCH_GET_FS = 0x1003 ; ARCH_SET_FS = 0x1002
  SYS_mmap = 9 ; PROT_RW = 3 ; MAP_PRIVATE|ANON = 0x22 ; fd = -1
  // read current fsbase into a stack local via arch_prctl(ARCH_GET_FS, &local)
  // if local != 0 -> return (real TLS already set up: re-executed launch)
  // else: p = mmap(0, 4096, 3, 0x22, -1, 0); arch_prctl(ARCH_SET_FS, p+2048)
  ```
  Do NOT touch R14 (g) or BP. SYSCALL clobbers RCX/R11 (fine here).

- **arm64** (`setup_universal_tls_arm64.s`): the thread pointer is `TPIDR_EL0`.
  `MRS Rx, TPIDR_EL0`; if 0, `mmap` then `MSR TPIDR_EL0, Rx` (no syscall needed
  to set it). arm64 is cross-compile-verified only in this environment; mark it
  as such.

- **no-op** (`setup_universal_tls_noop.go`, build `!cgo && linux && !android &&
  !goffi_universal`): `func setupUniversalTLS() {}`.

- **call site:** in `internal/fakecgo/go_linux_amd64.go` and `go_linux_arm64.go`,
  make `setupUniversalTLS()` the first line of `x_cgo_init`, immediately before
  the existing `maybeReexecUniversal()` call.

### How to verify the fix
```
export PATH=$PATH:/usr/local/go/bin
bash scripts/build-universal.sh -o /tmp/uprobe ./cmd/universal-probe
# glibc host:
/tmp/uprobe                      # expect: ... UNIVERSAL-PROBE-OK
# musl: run the SAME /tmp/uprobe inside an Alpine userland (chroot/container)
#   expect: "info re-exec bridge active: true" then UNIVERSAL-PROBE-OK
```
Both must print `UNIVERSAL-PROBE-OK`. `cmd/universal-probe` already auto-detects
the host libc the same way the bridge does.

---

## 4. What is already done (file-by-file)

All of the following compiles under CGO_ENABLED=0 in every mode (default /
`goffi_universal` / `goffi_static`; `goffi_musl` with its usual
`-gcflags=github.com/go-webgpu/goffi/internal/dl=-std`). It is committed on
branch `feature/profile-u-musl` as two WIP commits.

**Retagged to be mutually exclusive with `goffi_universal`** (added
`&& !goffi_universal` to their build constraints), so the universal empty-SONAME
files take over cleanly:
- `internal/dl/dl_linux_dynamic.go`, `internal/dl/dl_musl_amd64.go`,
  `internal/dl/dl_musl_arm64.go`
- `internal/syscall/errno_linux.go`, `internal/syscall/errno_musl_amd64.go`,
  `internal/syscall/errno_musl_arm64.go`
- `internal/fakecgo/symbols_linux.go`, `internal/fakecgo/symbols_musl_amd64.go`,
  `internal/fakecgo/symbols_musl_arm64.go`

**New — universal empty-SONAME imports (no DT_NEEDED):**
- `internal/dl/dl_universal.go` — `dlopen/dlsym/dlerror/dlclose` with `""`.
- `internal/syscall/errno_universal.go` — `__errno_location` with `""`.
- `internal/fakecgo/symbols_universal.go` — `malloc/free/setenv/unsetenv/`
  `sigfillset/nanosleep/abort/sigaltstack/pthread_*` with `""`. **Matches the
  musl symbol set** (i.e. **no** `pthread_get_stacksize_np`, which is Darwin-only
  and musl binds eagerly; it would fail to resolve).

**New — the re-exec bridge:**
- `internal/fakecgo/reexec_syscall_amd64.s`, `reexec_syscall_arm64.s` —
  `func rawsyscall6(trap,a1..a6 uintptr) (r1 uintptr)`, libc-free raw syscall.
- `internal/fakecgo/reexec_table_amd64.go`, `reexec_table_arm64.go` — per-arch
  syscall numbers (the `*at` forms only, so the same Go compiles on both arches)
  and the loader/libc table:
  - amd64: glibc `/lib64/ld-linux-x86-64.so.2` + `libc.so.6`; musl
    `/lib/ld-musl-x86_64.so.1` + `libc.musl-x86_64.so.1`.
  - arm64: glibc `/lib/ld-linux-aarch64.so.1` + `libc.so.6`; musl
    `/lib/ld-musl-aarch64.so.1` + `libc.musl-aarch64.so.1`.
- `internal/fakecgo/reexec_universal_linux.go` — the full nosplit,
  heap-free, libc-free re-exec: mmap scratch, read `/proc/self/environ` and
  `/proc/self/cmdline`, guard check (`GOFFI_UNIVERSAL_REEXEC=1`), discover the
  host loader (glibc first, musl fallback, via raw `faccessat`), build argv/envp
  pointer arrays (with a `cstr` bump-allocator that copies rodata strings +
  NUL into mmap), and `execve`. Contains a prominent **KNOWN ISSUE** comment
  describing the TLS blocker in §3.
- `internal/fakecgo/reexec_noop.go` — no-op `maybeReexecUniversal()` for
  non-universal linux builds.

**Changed — call site:**
- `internal/fakecgo/go_linux_amd64.go`, `go_linux_arm64.go` — call
  `maybeReexecUniversal()` at the very top of `x_cgo_init` (before `malloc`).
  *(The `setupUniversalTLS()` call from §3 must be added immediately before it.)*

**New — tooling:**
- `cmd/goffi-strip-interp/main.go` — flips `PT_INTERP` → `PT_NULL` (tiny,
  dependency-free ELF edit; handles ELFCLASS32/64).
- `cmd/universal-probe/main.go` — runtime FFI probe (LoadLibrary, sqrt, strlen,
  getpid, qsort callback, 32-goroutine thread hammer); auto-detects the host
  libc; prints `UNIVERSAL-PROBE-OK`.
- `scripts/build-universal.sh` — `go build -tags goffi_universal
  CGO_ENABLED=0` then strip the interpreter.

`ffi.Available()` returns `!static.Enabled`; universal is **not** static, so FFI
is enabled (`Available()==true`) automatically — no change needed there.

---

## 5. Remaining work (ordered, concrete)

1. **[DONE] `setupUniversalTLS` shim** (amd64 asm + arm64 asm + no-op stub +
   call site) — see §3. Landed; the first launch now sets up TLS and re-execs.

1b. **[DONE] glibc no-interp binding.** glibc's `ld.so` only binds a re-exec'd
   main object that carries a `PT_INTERP`; musl binds a no-interp binary
   directly. The bridge now branches on the detected libc: on musl it re-execs
   the on-disk binary as-is; on glibc it copies `/proc/self/exe` into a memfd
   with the `PT_INTERP` header restored and re-execs that. **Result:
   `UNIVERSAL-PROBE-OK` on both glibc and musl end-to-end (same binary,
   autonomous re-exec).** The universal probe's FP check was switched from
   `sqrt` (libm on glibc) to `atof` (libc on both).

2. **[DONE] `internal/loader` package** — the auditable "loader" port. A per-arch table
   (glibc/musl × amd64/arm64) of loader path + libc SONAME, plus runtime
   probing: detect the host libc flavor (filesystem existence of the loader
   paths; optionally read `PT_INTERP` from `/proc/self/exe` and/or auxv
   `AT_BASE`). Expose via the public `ffi` package:
   - `ffi.HostLoader() string`, `ffi.HostLibC() string`, `ffi.LibcKind()`
     (glibc/musl/unknown). Keep the table in sync with
     `reexec_table_*.go` (or have that file reference this package — but note the
     re-exec path is nosplit/libc-free and cannot import general code, so a small
     duplicated table there is acceptable; add a test asserting they match).

3. **[DONE] `cmd/goffi-audit`** — Profile-U contract checker: open a binary with
   `debug/elf` and assert it has **no `PT_INTERP`** and **no `DT_NEEDED`**. Exit
   non-zero otherwise. This is the portable essence of static-everywhere's
   onebin `c_profile.c`.

4. **[DONE] purego coexistence (CGO_ENABLED=0)** — goffi already has a `nofakecgo`
   build tag (`ffi/fakecgo_unix.go` is `//go:build ... && !nofakecgo ...`) that
   drops goffi's fakecgo so purego's provides the cgo-runtime symbols. Verified:
   `-tags nofakecgo` makes goffi+purego build **and** run under CGO_ENABLED=0.
   Harden + document this, and add a CI job that builds and runs a tiny
   goffi+purego program with `-tags nofakecgo`. Document that the `goffi_universal`
   build owns the cgo runtime and must not be combined with purego's fakecgo
   (using `nofakecgo` there would disable the re-exec bridge).

5. **[DONE] Tests:**
   - `internal/loader` unit tests (table correctness; flavor detection).
   - `ffi/universal_link_test.go`: build a `-tags goffi_universal` binary and
     assert **no `DT_NEEDED`** via `debug/elf` (interp-strip is a separate build
     step; the audit tool covers the interp check).
   - Confirm the existing `ffi/musl_directives_test.go` (`TestMuslDirectiveParity`)
     still passes; extend only if it needs to know about the universal files.

6. **[DONE] CI** — landed as a separate `.github/workflows/universal.yml`
   rather than inside `ci.yml`, mirroring the existing musl pattern
   (`scripts/check-musl.sh` + `cmd/musl-probe`). Add:
   - a `universal-build` job that builds the probe, strips the interp, and
     asserts the ELF contract (no PT_INTERP, no DT_NEEDED);
   - runs of `cmd/universal-probe` under **both** an Alpine (musl) container and
     a Debian/Ubuntu (glibc) container — the *same* built binary in both;
   - a `goffi-audit` job;
   - a `purego-coexistence` job (CGO_ENABLED=0, `-tags nofakecgo`).
   Existing CI pins a specific Go toolchain; the universal build needs **no**
   `-gcflags` (empty-SONAME `cgo_import_dynamic` is allowed in non-cgo code; only
   `//go:cgo_dynamic_linker`, which universal does not use, was restricted).
   Landed with all four jobs. Two notes for anyone editing that workflow:
   - it pins `go-version: '1.26.x'` like `ci.yml`, because the `goffi_musl`
     build's `-gcflags=...=-std` workaround is calibrated against that toolchain;
   - it runs `go vet -unsafeptr=false`. A bare `go vet ./...` fails on
     `ffi/callback_pointer.go` (the deliberate `//go:nocheckptr` helper that
     rebuilds a pointer from a native callback register). `ci.yml` never saw
     this because it runs vet through golangci-lint, whose `.golangci.yml`
     excludes `govet` on exactly those paths; the flag is the bare-vet
     equivalent of that exclusion. Do not "fix" it by rewriting the helper.

7. **[DONE] Docs + attribution:**
   - Update `docs/MUSL.md` (mention the universal build as the single-binary
     alternative).
   - New `docs/PROFILE_U.md` (user-facing): how to build a universal binary, the
     runtime re-exec behavior, and the limitations (Linux-only; a system whose
     loader we do not recognise cannot do FFI; argv[0] becomes the resolved exe
     path after re-exec).
   - Attribute `unxed/static-everywhere` and `pg83/solo` in `NOTICE`/`README`.
   `docs/PROFILE_U.md` also records the purego/pureffi situation: `-tags
   nofakecgo` cannot be used in universal mode (it removes the re-exec bridge
   along with fakecgo), so a purego-API dependency should go through
   `unxed/pureffi`, which carries no fakecgo of its own. pureffi needs **no**
   adaptation for this branch: Profile U only adds build-tag-gated files plus
   the new `ffi.HostLoader`/`HostLibC`/`LibcKind` helpers, and pureffi touches
   neither.

---

## 6. How to build and test (reference)

```
export PATH=$PATH:/usr/local/go/bin

# Compile checks (all must pass):
CGO_ENABLED=0 go build ./...                                   # default (glibc)
CGO_ENABLED=0 go build -tags goffi_universal ./...             # universal
CGO_ENABLED=0 go build -tags goffi_static ./...                # static (FFI off)
CGO_ENABLED=0 go build -tags goffi_musl \
  -gcflags=github.com/go-webgpu/goffi/internal/dl=-std ./...   # musl

# Build + strip a universal binary:
bash scripts/build-universal.sh -o /tmp/uprobe ./cmd/universal-probe

# ELF contract (after strip): expect NO interpreter, NO NEEDED:
readelf -lW /tmp/uprobe | grep -i "Requesting"    # -> nothing
readelf -dW /tmp/uprobe | grep NEEDED             # -> nothing

# Run on glibc (the dev host) and on musl (Alpine userland via chroot/container):
/tmp/uprobe                                        # -> UNIVERSAL-PROBE-OK (after §3 fix)

# purego coexistence (CGO-free):
#   build a small program importing both goffi and purego with -tags nofakecgo,
#   CGO_ENABLED=0, and run it.
```

Notes for the dev sandbox specifically (may not exist in other environments):
Go is at `/usr/local/go`; an Alpine musl rootfs was extracted at
`/tmp/u/rootfs` (loader `/tmp/u/rootfs/lib/ld-musl-x86_64.so.1`), usable via
`chroot /tmp/u/rootfs …`. In CI, use an official Alpine container instead.

---

## 7. Honesty notes / decisions (do not silently reverse)

- **No SoLo.** A full in-process foreign-libc loader is intentionally *not*
  vendored, matching static-everywhere's own scoping. The universal build reuses
  the host's own `ld.so` via `--preload`.
- **Linux-only.** The universal mode targets Linux. On a system whose dynamic
  loader we do not recognise, FFI is impossible (the process cannot bind libc);
  that is a documented limitation. A program that needs FFI on such a system
  cannot run in universal mode.
- **arm64 is cross-compile-verified only** in this environment; state that in
  docs/commits. amd64 is the run-tested arch.
- **purego coexistence is guaranteed for the default/musl/static modes**
  (CGO_ENABLED=0, `-tags nofakecgo`). The universal build owns the cgo runtime
  and should not be combined with purego's fakecgo.
- **argv[0]** of a universal binary becomes the resolved executable path after
  the re-exec (we pass `/proc/self/exe` or its readlink target as the program).
  Acceptable; document it.

---

## 8. Delivery / pushing

Development happened in a sandbox with **no GitHub credentials** (the fork remote
is read-only over HTTPS there), so commits are handed off as `git format-patch`
series (and/or a repo tarball) to be applied and pushed by the maintainer:

```
git checkout -b feature/profile-u-musl        # base: origin/main
git am 0001-*.patch 0002-*.patch [0003-...]   # in order
git push -u origin feature/profile-u-musl
```

From here on, each hand-off is an **incremental** patch (only the delta over the
previous commit).

---

## START HERE (immediate next action)

Everything in §5 has landed. What remains is verification and release hygiene:

1. **Watch the `universal` workflow go green.** Until the vet fix above, the
   build job died before `go test`, so the unit-test and both-libc run jobs on
   this branch are still **unverified in CI**. Read that run before trusting
   the tests.
2. **arm64 is still cross-compile-verified only.** Run `cmd/universal-probe` on
   a real aarch64 glibc host and an aarch64 Alpine host, then drop the caveat
   from `docs/PROFILE_U.md` and this file.
3. **Release notes.** `CHANGELOG.md` / `ROADMAP.md` have no Profile U entry yet;
   add one when the branch is proposed upstream.

Quick check that the mechanism still works after any change:
`scripts/build-universal.sh -o /tmp/uprobe ./cmd/universal-probe`, then run
`/tmp/uprobe` on glibc and inside an Alpine (musl) userland with `/proc`
mounted; both must print `UNIVERSAL-PROBE-OK`.
