package types

import (
	"errors"
	"runtime"
)

// RuntimeEnvironment returns current runtime OS and architecture
func RuntimeEnvironment() (os, arch string) {
	return runtime.GOOS, runtime.GOARCH
}

// CallingConvention represents function calling conventions
type CallingConvention int

const (
	UnixCallingConvention CallingConvention = iota + 1
	WindowsCallingConvention
	GnuWindowsCallingConvention
)

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

// TypeDescriptor describes a data type
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
