# ADR-001: Userspace ELF loader for static binaries

**Status:** Proposed (research / preview)  
**Date:** 2026-09-08  
**Tracking:** [goffi#74](https://github.com/go-webgpu/goffi/issues/74), [gogpu#474](https://github.com/gogpu/gogpu/issues/474)  
**Go baseline:** 1.25+ (`structs.HostLayout`, `CGO_ENABLED=0`)

## Context

Default goffi Linux builds use `//go:cgo_import_dynamic` to resolve `dlopen` /
`dlsym` from `libdl.so.2`. That forces `PT_INTERP` + `DT_NEEDED`, so binaries
are dynamically linked even with `CGO_ENABLED=0`.

`-tags goffi_static` removes those imports and yields a fully static ELF, but
`ffi.LoadLibrary` returns `ErrStaticBuild`. Host `dlopen` cannot work inside a
static ELF: the kernel only loads `ld.so` when `PT_INTERP` is present (SunOS
userspace-loader design). Windows differs because `ntdll` is always mapped.

Consumer apps (e.g. f4) want **static distribution** and, ideally, still load
system `.so` files (Wayland, Vulkan ICD, libX11) at runtime.

## Decision

Explore a **pure-Go userspace ELF `.so` loader** gated behind
`-tags goffi_elfloader` (experimental, not default). It is the only credible
path to “static binary + runtime LoadLibrary-like behavior” on Linux without
shipping `ld.so`.

Ship Track 1–2 (`goffi_static` + docs/CI) first. This ADR does **not** block
closing the documentation / static-profile side of #74 / gogpu#474.

## Goals

1. Map `ET_DYN` shared objects via `unix.Mmap` / `PROT_EXEC` without calling
   host `dlopen`.
2. Apply relative / symbolic relocations needed for a small allowlisted set of
   UI/GPU libraries.
3. Hand resolved symbol addresses to existing goffi `CallFunction` (no change
   to the assembly call path once the function pointer is known).
4. Fail closed on TLS, IFUNC, GNU symbol versioning, and glibc-private ABI
   that cannot be reproduced safely.

## Non-goals

- Full glibc / musl loader compatibility.
- Loading arbitrary untrusted `.so` from the network.
- Default-on behavior for release builds.
- Mach-O / PE loaders in v0 (Linux amd64 first, then arm64).

## Proposed design (v0)

```
LoadLibrary(path)
  → open + read ELF headers
  → mmap PT_LOAD segments (honor p_align)
  → apply R_*_RELATIVE / R_*_GLOB_DAT against local + allowlisted modules
  → return handle; GetSymbol walks .dynsym / hash
CallFunction(cif, sym, ...)  // unchanged
```

Build tag: `goffi_elfloader` (implies or composes with static profile).  
Public API stays `ffi.LoadLibrary` / `GetSymbol`; implementation switches
under the tag.

Threat model: load **system libraries by absolute path** only
(`/usr/lib/**`, `/lib/**`). Document dual-use concerns of in-memory loaders;
reject relative paths and `LD_LIBRARY_PATH` search in v0.

## Alternatives

| Alternative | Why not primary |
|-------------|-----------------|
| Document-only | Does not meet static + FFI consumer need |
| musl-dynamic Alpine | Valid container story; not `FROM scratch` |
| Embed `ld.so` | Not Pure Go; redistributes glibc/musl loader |
| External linker `-static` | Fights CGO_ENABLED=0; still no useful `dlopen` |

## Consequences

- Large engineering surface (relocations, TLS, IFUNC).
- Security review required before preview announcement.
- Success metric for spike: load a trivial self-built `.so` and call one
  exported `int add(int,int)` via `CallFunction` from a static binary.

## Spike checklist

- [ ] Linux amd64: map toy `.so`, resolve one symbol, call via goffi
- [ ] Reject TLS / IFUNC with typed errors
- [ ] Absolute-path allowlist
- [ ] Document Go/No-Go for arm64 preview
