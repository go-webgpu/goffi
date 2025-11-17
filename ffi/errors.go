package ffi

import (
	"fmt"
)

// InvalidCallInterfaceError indicates CallInterface preparation failed due to
// invalid parameters.
//
// This error provides detailed information about which field failed validation
// and why, enabling programmatic error handling and better debugging.
//
// Example:
//
//	var icErr *InvalidCallInterfaceError
//	if errors.As(err, &icErr) {
//	    fmt.Printf("Field %s failed: %s\n", icErr.Field, icErr.Reason)
//	    if icErr.Index >= 0 {
//	        fmt.Printf("At index: %d\n", icErr.Index)
//	    }
//	}
type InvalidCallInterfaceError struct {
	Field  string // Which field was invalid ("cif", "returnType", "argTypes", etc.)
	Reason string // Why it was invalid (human-readable description)
	Index  int    // For array fields like argTypes (-1 if not applicable)
}

func (e *InvalidCallInterfaceError) Error() string {
	if e.Index >= 0 {
		return fmt.Sprintf("invalid call interface: %s[%d]: %s",
			e.Field, e.Index, e.Reason)
	}
	return fmt.Sprintf("invalid call interface: %s: %s", e.Field, e.Reason)
}

// Is implements error equality for errors.Is().
func (e *InvalidCallInterfaceError) Is(target error) bool {
	_, ok := target.(*InvalidCallInterfaceError)
	return ok
}

// UnsupportedPlatformError indicates the current platform is not supported by FFI.
//
// This error is returned when attempting to use FFI on a platform that doesn't
// have an implementation (e.g., ARM64 before it's fully implemented).
//
// Example:
//
//	var upErr *UnsupportedPlatformError
//	if errors.As(err, &upErr) {
//	    fmt.Printf("Platform %s/%s not supported\n", upErr.OS, upErr.Arch)
//	}
type UnsupportedPlatformError struct {
	OS   string // Operating system (e.g., "linux", "windows", "darwin")
	Arch string // Architecture (e.g., "amd64", "arm64")
}

func (e *UnsupportedPlatformError) Error() string {
	return fmt.Sprintf("unsupported platform: %s/%s (FFI not implemented for this platform)",
		e.OS, e.Arch)
}

// Is implements error equality for errors.Is().
func (e *UnsupportedPlatformError) Is(target error) bool {
	_, ok := target.(*UnsupportedPlatformError)
	return ok
}

// LibraryError wraps dynamic library loading and symbol resolution errors.
//
// This error provides context about which library operation failed and includes
// the underlying OS-specific error for detailed diagnostics.
//
// Example:
//
//	var libErr *LibraryError
//	if errors.As(err, &libErr) {
//	    fmt.Printf("Failed to %s library %q\n", libErr.Operation, libErr.Name)
//	    fmt.Printf("OS error: %v\n", libErr.Err)
//	}
type LibraryError struct {
	Operation string // "load", "symbol", or "free"
	Name      string // Library path or symbol name
	Err       error  // Underlying OS error (can be nil)
}

func (e *LibraryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("library %s failed for %q: %v", e.Operation, e.Name, e.Err)
	}
	return fmt.Sprintf("library %s failed for %q", e.Operation, e.Name)
}

// Unwrap returns the underlying error for errors.Unwrap().
func (e *LibraryError) Unwrap() error {
	return e.Err
}

// Is implements error equality for errors.Is().
func (e *LibraryError) Is(target error) bool {
	_, ok := target.(*LibraryError)
	return ok
}

// CallingConventionError indicates an unsupported or invalid calling convention.
//
// This error is returned when attempting to use a calling convention that is
// not supported on the current platform or is invalid.
type CallingConventionError struct {
	Convention int    // The invalid convention value
	Platform   string // Current platform (OS/Arch)
	Reason     string // Why it's not supported
}

func (e *CallingConventionError) Error() string {
	return fmt.Sprintf("unsupported calling convention %d on %s: %s",
		e.Convention, e.Platform, e.Reason)
}

// Is implements error equality for errors.Is().
func (e *CallingConventionError) Is(target error) bool {
	_, ok := target.(*CallingConventionError)
	return ok
}

// TypeValidationError indicates a type descriptor failed validation.
//
// This error provides details about which type failed and why, helping users
// fix type definition issues.
type TypeValidationError struct {
	TypeName string // Name or description of the type
	Kind     int    // The TypeKind value that failed
	Reason   string // Why validation failed
	Index    int    // For composite types (-1 if not applicable)
}

func (e *TypeValidationError) Error() string {
	if e.Index >= 0 {
		return fmt.Sprintf("type validation failed for %s[%d] (kind=%d): %s",
			e.TypeName, e.Index, e.Kind, e.Reason)
	}
	if e.TypeName != "" {
		return fmt.Sprintf("type validation failed for %s (kind=%d): %s",
			e.TypeName, e.Kind, e.Reason)
	}
	return fmt.Sprintf("type validation failed (kind=%d): %s", e.Kind, e.Reason)
}

// Is implements error equality for errors.Is().
func (e *TypeValidationError) Is(target error) bool {
	_, ok := target.(*TypeValidationError)
	return ok
}

// Deprecated: Legacy sentinel errors kept for backwards compatibility.
// Use typed errors above with errors.As() for better error handling.
var (
	// ErrInvalidCallInterface is deprecated. Use InvalidCallInterfaceError instead.
	//
	// Deprecated: This sentinel error is kept for backwards compatibility only.
	// New code should use errors.As() with *InvalidCallInterfaceError to get
	// detailed error information.
	ErrInvalidCallInterface = &InvalidCallInterfaceError{
		Field:  "unknown",
		Reason: "invalid call interface",
		Index:  -1,
	}

	// ErrFunctionCallFailed is deprecated. Use specific typed errors instead.
	//
	// Deprecated: This generic error doesn't provide useful debugging information.
	// New code should handle specific error types returned by CallFunction.
	ErrFunctionCallFailed = fmt.Errorf("function call failed")
)

// Helper functions for creating common errors

// newInvalidTypeError creates a TypeValidationError for an invalid type kind.
func newInvalidTypeError(typeName string, kind int, reason string) error {
	return &TypeValidationError{
		TypeName: typeName,
		Kind:     kind,
		Reason:   reason,
		Index:    -1,
	}
}

// newInvalidTypeAtIndexError creates a TypeValidationError for a type at a specific index.
func newInvalidTypeAtIndexError(typeName string, kind int, index int, reason string) error {
	return &TypeValidationError{
		TypeName: typeName,
		Kind:     kind,
		Reason:   reason,
		Index:    index,
	}
}
