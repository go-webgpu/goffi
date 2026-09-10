//go:build goffi_static && ((linux && !android) || darwin || freebsd || android) && (amd64 || arm64)

package syscall

// ErrnoFnAddr returns 0 under -tags goffi_static.
//
// The assembly trampoline skips errno capture when errnoFn is 0 (same as the
// Windows path). Dynamic imports for __errno_location / __error are excluded
// from static builds so the binary stays free of DT_NEEDED entries.
func ErrnoFnAddr() uintptr {
	return 0
}
