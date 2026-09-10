//go:build goffi_static && (linux || darwin || freebsd) && !cgo

package ffi

// Pull in the static-profile fakecgo stubs (crosscall2 abort trampoline) so the
// ffi package links when callback assembly is present. Dynamic loading remains
// unavailable; see ErrStaticBuild.
import _ "github.com/go-webgpu/goffi/internal/fakecgo"
