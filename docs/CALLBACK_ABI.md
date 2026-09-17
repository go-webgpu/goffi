# Callback argument support, per platform

`ffi.NewCallback` turns a Go function into a C function pointer. What that
function may look like is narrower than what the calling direction supports,
and it is not the same everywhere. This page says where the limits are and why,
so a caller can tell an unimplemented case from a broken one.

## What is supported today

| | integers, pointers, bool | float32 / float64 | struct by value |
|---|---|---|---|
| linux / darwin / freebsd, **amd64** | yes | yes | **yes** |
| linux / darwin / freebsd, **arm64** | yes | yes | **no** — panics |
| **windows** (amd64, arm64) | uintptr-sized only | no | no |
| **android** arm64 | rejected by design (see `docs/ANDROID.md`) | | |

Return values follow the same rows: amd64 and arm64 return integers, pointers,
bool and floats; Windows returns exactly one uintptr-sized value.

Windows is not a gap to close. `ffi.NewCallback` there delegates to Go's
`syscall.NewCallback`, which accepts only uintptr-sized arguments and one
uintptr-sized result; that is the whole contract the platform offers without
cgo, and `purego` documents the same one.

## The arm64 gap

On arm64 `validateCallbackSignature` (`ffi/callback_arm64.go`) rejects any
struct argument:

```
panic: ffi: unsupported callback argument type: struct
```

The amd64 dispatcher classifies aggregates per the System V ABI — one eightbyte
or two, INTEGER or SSE per eightbyte, MEMORY above 16 bytes — in
`callbackWrap` (`ffi/callback.go`), with `classifyEightbyte` and
`isStructAllFloats` doing the work. The arm64 dispatcher has no equivalent: it
reads each argument from one D or X register slot, or one stack slot, and an
aggregate does not fit that shape.

This is missing work, not a platform limit. AAPCS64 defines the classification
and nothing about it is out of reach:

- **HFA / HVA** — an aggregate whose members are all the same floating-point
  type, at most four of them, goes in that many consecutive V registers. A
  struct of four `float32` is the awkward case: the members occupy S0–S3, i.e.
  the low 32 bits of four consecutive V registers, while the dispatcher's frame
  stores each V register as a 64-bit slot.
- **Small aggregates (≤ 16 bytes)** — passed in one or two X registers,
  as if the struct were copied into 8-byte chunks.
- **Larger aggregates** — passed indirectly: the caller places a copy in memory
  and passes its address in one X register.
- **Register exhaustion** — when the required registers are not all available,
  the whole aggregate goes on the stack, and the remaining registers are *not*
  backfilled by later arguments.
- **Apple** — darwin/arm64 packs stack arguments to their natural alignment
  rather than 8-byte slots, so a stack-passed aggregate needs its own handling
  there. Any implementation has to be checked against both a Linux and an Apple
  toolchain, not one of them.

The calling direction (`ffi.CallFunction` with a struct argument) already does
classify aggregates on arm64; only the callback direction is missing.

## Working around it

Take a pointer instead of a value. `func(r *Rect)` is one X register on every
platform in the table, costs nothing, and is what most C APIs that hand a
struct to a callback do anyway.

## If you are implementing this

`ffi/callback_struct_args_test.go` is the amd64 model: it drives `callbackWrap`
directly with a hand-built argument frame, so the classification can be tested
without a C toolchain and without a real foreign call. An arm64 counterpart
building an AAPCS64 frame is the way to start, with the four-`float32` HFA and
a > 16-byte aggregate as the first two cases. `unxed/pureffi`'s
`TestFullCircleFFIStruct` is a ready end-to-end check: it skips on arm64 today
and would start running.
