//go:build (linux || darwin || freebsd) && !cgo && !nofakecgo

package ffi

// fakecgo enables runtime.cgocall without CGO_ENABLED=1.
// This allows goffi's FFI implementation to work safely on Unix-like systems.
//
// The fakecgo package injects runtime.iscgo=true and installs function pointers
// (_cgo_init, _cgo_thread_start, etc.) that the Go runtime requires for cgocall.
//
// Build tag "nofakecgo" disables this import for projects that already provide
// these symbols through another package (e.g., purego's internal/fakecgo).
// Without this tag, linking both goffi and purego with CGO_ENABLED=0 causes:
//   link: duplicated definition of symbol _cgo_init
//
// Usage:
//   go build -tags nofakecgo ./...   # when purego is also in the dependency tree

import (
	_ "github.com/go-webgpu/goffi/internal/fakecgo"
)
