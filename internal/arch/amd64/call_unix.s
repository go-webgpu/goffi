//go:build amd64 && (linux || darwin)

// This file is intentionally empty.
// The previous callUnix64 experiment was removed in v0.4.1 (TASK-022, GAP-11).
// The syscall layer now uses internal/syscall/syscall_unix_amd64.s exclusively.
