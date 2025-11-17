package ffi

import (
	"errors"
	"testing"

	"github.com/go-webgpu/goffi/types"
)

// TestTypedErrors demonstrates how to use typed errors with errors.As().
func TestTypedErrors(t *testing.T) {
	t.Run("InvalidCallInterfaceError", func(t *testing.T) {
		// Test nil cif
		err := PrepareCallInterface(nil, types.DefaultCall, types.VoidTypeDescriptor, nil)

		var icErr *InvalidCallInterfaceError
		if !errors.As(err, &icErr) {
			t.Fatalf("Expected InvalidCallInterfaceError, got %T", err)
		}

		if icErr.Field != "cif" {
			t.Errorf("Expected Field='cif', got '%s'", icErr.Field)
		}
		if icErr.Reason != "must not be nil" {
			t.Errorf("Expected Reason='must not be nil', got '%s'", icErr.Reason)
		}
		if icErr.Index != -1 {
			t.Errorf("Expected Index=-1, got %d", icErr.Index)
		}

		// Test error message format
		expectedMsg := "invalid call interface: cif: must not be nil"
		if icErr.Error() != expectedMsg {
			t.Errorf("Expected message '%s', got '%s'", expectedMsg, icErr.Error())
		}
	})

	t.Run("InvalidCallInterfaceError_ReturnType", func(t *testing.T) {
		// Test nil returnType
		cif := &types.CallInterface{}
		err := PrepareCallInterface(cif, types.DefaultCall, nil, nil)

		var icErr *InvalidCallInterfaceError
		if !errors.As(err, &icErr) {
			t.Fatalf("Expected InvalidCallInterfaceError, got %T", err)
		}

		if icErr.Field != "returnType" {
			t.Errorf("Expected Field='returnType', got '%s'", icErr.Field)
		}
	})

	t.Run("LibraryError_Load", func(t *testing.T) {
		// Test loading non-existent library
		_, err := LoadLibrary("nonexistent_library_12345.dll")

		var libErr *LibraryError
		if !errors.As(err, &libErr) {
			t.Fatalf("Expected LibraryError, got %T", err)
		}

		if libErr.Operation != "load" {
			t.Errorf("Expected Operation='load', got '%s'", libErr.Operation)
		}
		if libErr.Name != "nonexistent_library_12345.dll" {
			t.Errorf("Expected Name='nonexistent_library_12345.dll', got '%s'", libErr.Name)
		}
		if libErr.Err == nil {
			t.Error("Expected underlying error, got nil")
		}

		// Test error unwrapping
		if errors.Unwrap(err) == nil {
			t.Error("Expected error to be unwrappable")
		}
	})

	t.Run("CallingConventionError", func(t *testing.T) {
		// Test invalid calling convention
		cif := &types.CallInterface{}
		invalidConvention := types.CallingConvention(99)
		err := PrepareCallInterface(cif, invalidConvention, types.VoidTypeDescriptor, nil)

		var ccErr *CallingConventionError
		if !errors.As(err, &ccErr) {
			t.Fatalf("Expected CallingConventionError, got %T: %v", err, err)
		}

		if ccErr.Convention != 99 {
			t.Errorf("Expected Convention=99, got %d", ccErr.Convention)
		}
	})

	t.Run("TypeValidationError", func(t *testing.T) {
		// Test invalid type kind
		cif := &types.CallInterface{}
		invalidType := &types.TypeDescriptor{
			Size:      4,
			Alignment: 4,
			Kind:      types.TypeKind(999), // Invalid kind
		}

		err := PrepareCallInterface(cif, types.DefaultCall, invalidType, nil)

		var tvErr *TypeValidationError
		if !errors.As(err, &tvErr) {
			t.Fatalf("Expected TypeValidationError, got %T: %v", err, err)
		}

		if tvErr.Kind != 999 {
			t.Errorf("Expected Kind=999, got %d", tvErr.Kind)
		}
	})

	t.Run("ErrorIs", func(t *testing.T) {
		// Test errors.Is() works with all typed errors

		// InvalidCallInterfaceError
		err1 := &InvalidCallInterfaceError{Field: "test", Reason: "test reason", Index: -1}
		if !errors.Is(err1, &InvalidCallInterfaceError{}) {
			t.Error("errors.Is() should work with InvalidCallInterfaceError")
		}

		// UnsupportedPlatformError
		err2 := &UnsupportedPlatformError{OS: "plan9", Arch: "386"}
		if !errors.Is(err2, &UnsupportedPlatformError{}) {
			t.Error("errors.Is() should work with UnsupportedPlatformError")
		}

		// LibraryError
		err3 := &LibraryError{Operation: "load", Name: "test.dll", Err: nil}
		if !errors.Is(err3, &LibraryError{}) {
			t.Error("errors.Is() should work with LibraryError")
		}

		// CallingConventionError
		err4 := &CallingConventionError{Convention: 99, Platform: "test/test", Reason: "test"}
		if !errors.Is(err4, &CallingConventionError{}) {
			t.Error("errors.Is() should work with CallingConventionError")
		}

		// TypeValidationError
		err5 := &TypeValidationError{TypeName: "test", Kind: 999, Reason: "test", Index: -1}
		if !errors.Is(err5, &TypeValidationError{}) {
			t.Error("errors.Is() should work with TypeValidationError")
		}
	})
}

// TestDeprecatedErrors ensures backwards compatibility with old sentinel errors.
func TestDeprecatedErrors(t *testing.T) {
	t.Run("ErrInvalidCallInterface", func(t *testing.T) {
		if ErrInvalidCallInterface == nil {
			t.Error("ErrInvalidCallInterface should exist for backwards compatibility")
		}

		// Should be of type *InvalidCallInterfaceError
		var icErr *InvalidCallInterfaceError
		if !errors.As(ErrInvalidCallInterface, &icErr) {
			t.Error("ErrInvalidCallInterface should be *InvalidCallInterfaceError")
		}
	})

	t.Run("ErrFunctionCallFailed", func(t *testing.T) {
		if ErrFunctionCallFailed == nil {
			t.Error("ErrFunctionCallFailed should exist for backwards compatibility")
		}
	})
}

// TestErrorMessages ensures error messages are informative.
func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name: "InvalidCallInterfaceError with index",
			err: &InvalidCallInterfaceError{
				Field:  "argTypes",
				Reason: "invalid type",
				Index:  2,
			},
			expected: "invalid call interface: argTypes[2]: invalid type",
		},
		{
			name: "InvalidCallInterfaceError without index",
			err: &InvalidCallInterfaceError{
				Field:  "cif",
				Reason: "is nil",
				Index:  -1,
			},
			expected: "invalid call interface: cif: is nil",
		},
		{
			name: "LibraryError with underlying error",
			err: &LibraryError{
				Operation: "load",
				Name:      "test.dll",
				Err:       errors.New("file not found"),
			},
			expected: "library load failed for \"test.dll\": file not found",
		},
		{
			name: "LibraryError without underlying error",
			err: &LibraryError{
				Operation: "load",
				Name:      "test.dll",
				Err:       nil,
			},
			expected: "library load failed for \"test.dll\"",
		},
		{
			name: "UnsupportedPlatformError",
			err: &UnsupportedPlatformError{
				OS:   "plan9",
				Arch: "386",
			},
			expected: "unsupported platform: plan9/386 (FFI not implemented for this platform)",
		},
		{
			name: "CallingConventionError",
			err: &CallingConventionError{
				Convention: 99,
				Platform:   "windows/amd64",
				Reason:     "invalid value",
			},
			expected: "unsupported calling convention 99 on windows/amd64: invalid value",
		},
		{
			name: "TypeValidationError with index",
			err: &TypeValidationError{
				TypeName: "structMember",
				Kind:     999,
				Reason:   "invalid",
				Index:    3,
			},
			expected: "type validation failed for structMember[3] (kind=999): invalid",
		},
		{
			name: "TypeValidationError without index",
			err: &TypeValidationError{
				TypeName: "returnType",
				Kind:     999,
				Reason:   "invalid",
				Index:    -1,
			},
			expected: "type validation failed for returnType (kind=999): invalid",
		},
		{
			name: "TypeValidationError without type name",
			err: &TypeValidationError{
				TypeName: "",
				Kind:     999,
				Reason:   "invalid",
				Index:    -1,
			},
			expected: "type validation failed (kind=999): invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.err.Error()
			if errMsg == "" {
				t.Error("Error message should not be empty")
			}
			if errMsg != tt.expected {
				t.Errorf("Expected message '%s', got '%s'", tt.expected, errMsg)
			}
		})
	}
}
