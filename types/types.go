package types

import (
	"errors"
	"runtime"
)

// RuntimeEnvironment returns current runtime OS and architecture
func RuntimeEnvironment() (os, arch string) {
	return runtime.GOOS, runtime.GOARCH
}

// CallingConvention represents function calling conventions used by different platforms.
type CallingConvention int

const (
	// UnixCallingConvention is the System V AMD64 ABI used on Linux, macOS, FreeBSD.
	UnixCallingConvention CallingConvention = iota + 1

	// WindowsCallingConvention is the Win64 (Microsoft x64) calling convention.
	WindowsCallingConvention

	// GnuWindowsCallingConvention is the GNU extension to Windows calling convention.
	GnuWindowsCallingConvention

	// Convenient aliases for common usage
	// DefaultCall automatically selects the platform's native calling convention.
	DefaultCall = CallingConvention(0) // Will be resolved by DefaultConvention()

	// CDecl is an alias for the C calling convention on the current platform.
	CDecl = UnixCallingConvention // Updated by init() on Windows

	// StdCall is Windows-specific (same as WindowsCallingConvention).
	StdCall = WindowsCallingConvention
)

// DefaultConvention returns the native calling convention for the current platform.
//
// Returns:
//   - UnixCallingConvention on Linux, macOS, FreeBSD
//   - WindowsCallingConvention on Windows
//
// Example:
//
//	convention := types.DefaultConvention()
//	// or just use types.DefaultCall constant
func DefaultConvention() CallingConvention {
	if runtime.GOOS == "windows" {
		return WindowsCallingConvention
	}
	return UnixCallingConvention
}

// TypeKind defines data type categories
type TypeKind int

const (
	VoidType TypeKind = iota
	IntType
	FloatType
	DoubleType
	UInt8Type
	SInt8Type
	UInt16Type
	SInt16Type
	UInt32Type
	SInt32Type
	UInt64Type
	SInt64Type
	StructType
	PointerType
)

// TypeDescriptor describes FFI type characteristics
type TypeDescriptor struct {
	Size      uintptr           // Size in bytes
	Alignment uintptr           // Alignment requirement
	Kind      TypeKind          // Type category
	Members   []*TypeDescriptor // For composite types
}

// Predefined type descriptors
var (
	VoidTypeDescriptor    = &TypeDescriptor{Size: 1, Alignment: 1, Kind: VoidType}
	IntTypeDescriptor     = &TypeDescriptor{Size: 4, Alignment: 4, Kind: IntType}
	FloatTypeDescriptor   = &TypeDescriptor{Size: 4, Alignment: 4, Kind: FloatType}
	DoubleTypeDescriptor  = &TypeDescriptor{Size: 8, Alignment: 8, Kind: DoubleType}
	UInt8TypeDescriptor   = &TypeDescriptor{Size: 1, Alignment: 1, Kind: UInt8Type}
	SInt8TypeDescriptor   = &TypeDescriptor{Size: 1, Alignment: 1, Kind: SInt8Type}
	UInt16TypeDescriptor  = &TypeDescriptor{Size: 2, Alignment: 2, Kind: UInt16Type}
	SInt16TypeDescriptor  = &TypeDescriptor{Size: 2, Alignment: 2, Kind: SInt16Type}
	UInt32TypeDescriptor  = &TypeDescriptor{Size: 4, Alignment: 4, Kind: UInt32Type}
	SInt32TypeDescriptor  = &TypeDescriptor{Size: 4, Alignment: 4, Kind: SInt32Type}
	UInt64TypeDescriptor  = &TypeDescriptor{Size: 8, Alignment: 8, Kind: UInt64Type}
	SInt64TypeDescriptor  = &TypeDescriptor{Size: 8, Alignment: 8, Kind: SInt64Type}
	PointerTypeDescriptor = &TypeDescriptor{Size: 8, Alignment: 8, Kind: PointerType}
)

// CallInterface represents a prepared function call interface
type CallInterface struct {
	Convention CallingConvention
	ArgCount   int
	ArgTypes   []*TypeDescriptor
	ReturnType *TypeDescriptor
	Flags      int     // Return flags
	StackBytes uintptr // Required stack space
}

// Return flags constants
const (
	ReturnVoid       = 0
	ReturnUInt8      = 1
	ReturnUInt16     = 2
	ReturnUInt32     = 3
	ReturnSInt8      = 4
	ReturnSInt16     = 5
	ReturnSInt32     = 6
	ReturnInt64      = 7
	ReturnInXMM32    = 8
	ReturnInXMM64    = 9
	ReturnViaPointer = 1 << 10
)

// Error constants
var (
	ErrUnsupportedArchitecture      = errors.New("unsupported architecture")
	ErrUnsupportedCallingConvention = errors.New("unsupported calling convention")
	ErrInvalidTypeDefinition        = errors.New("invalid type definition")
	ErrUnsupportedReturnType        = errors.New("unsupported return type")
)
