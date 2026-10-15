//go:build goffi_static && ((linux && !android) || darwin || freebsd || (android && arm64)) && (amd64 || arm64)

package ffi

// staticBuild reports that this is a -tags goffi_static build on a platform
// where the tag takes effect (the same constraint as dl_static.go).
const staticBuild = true
