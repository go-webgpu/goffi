//go:build !(goffi_static && ((linux && !android) || darwin || freebsd || (android && arm64)) && (amd64 || arm64))

package ffi

// staticBuild is false: dynamic loading is compiled in. See static_mode.go.
const staticBuild = false
