//go:build windows

// Package syscall provides low-level FFI syscall infrastructure.
package syscall

// ErrnoFnAddr returns 0 on Windows because errno capture via the Unix
// __errno_location mechanism is not applicable. Windows uses GetLastError()
// for Win32 errors, which is already captured by syscall.SyscallN as the
// third return value. CRT errno is rarely used for Win32 APIs.
func ErrnoFnAddr() uintptr {
	return 0
}
