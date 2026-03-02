package types

import (
	"runtime"
	"testing"
)

func TestRuntimeEnvironment(t *testing.T) {
	os, arch := RuntimeEnvironment()
	if os != runtime.GOOS {
		t.Errorf("RuntimeEnvironment() os = %q, want %q", os, runtime.GOOS)
	}
	if arch != runtime.GOARCH {
		t.Errorf("RuntimeEnvironment() arch = %q, want %q", arch, runtime.GOARCH)
	}
}

func TestDefaultConvention(t *testing.T) {
	conv := DefaultConvention()
	if runtime.GOOS == "windows" {
		if conv != WindowsCallingConvention {
			t.Errorf("DefaultConvention() = %d, want WindowsCallingConvention", conv)
		}
	} else {
		if conv != UnixCallingConvention {
			t.Errorf("DefaultConvention() = %d, want UnixCallingConvention", conv)
		}
	}
}

func TestTypeDescriptors(t *testing.T) {
	tests := []struct {
		name      string
		desc      *TypeDescriptor
		wantSize  uintptr
		wantAlign uintptr
		wantKind  TypeKind
	}{
		{"Void", VoidTypeDescriptor, 1, 1, VoidType},
		{"Int", IntTypeDescriptor, 4, 4, IntType},
		{"Float", FloatTypeDescriptor, 4, 4, FloatType},
		{"Double", DoubleTypeDescriptor, 8, 8, DoubleType},
		{"UInt8", UInt8TypeDescriptor, 1, 1, UInt8Type},
		{"SInt8", SInt8TypeDescriptor, 1, 1, SInt8Type},
		{"UInt16", UInt16TypeDescriptor, 2, 2, UInt16Type},
		{"SInt16", SInt16TypeDescriptor, 2, 2, SInt16Type},
		{"UInt32", UInt32TypeDescriptor, 4, 4, UInt32Type},
		{"SInt32", SInt32TypeDescriptor, 4, 4, SInt32Type},
		{"UInt64", UInt64TypeDescriptor, 8, 8, UInt64Type},
		{"SInt64", SInt64TypeDescriptor, 8, 8, SInt64Type},
		{"Pointer", PointerTypeDescriptor, 8, 8, PointerType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.desc.Size != tt.wantSize {
				t.Errorf("Size = %d, want %d", tt.desc.Size, tt.wantSize)
			}
			if tt.desc.Alignment != tt.wantAlign {
				t.Errorf("Alignment = %d, want %d", tt.desc.Alignment, tt.wantAlign)
			}
			if tt.desc.Kind != tt.wantKind {
				t.Errorf("Kind = %d, want %d", tt.desc.Kind, tt.wantKind)
			}
		})
	}
}

func TestCallInterfaceZeroValue(t *testing.T) {
	var cif CallInterface
	if cif.Convention != 0 {
		t.Errorf("zero CallInterface.Convention = %d, want 0", cif.Convention)
	}
	if cif.ArgCount != 0 {
		t.Errorf("zero CallInterface.ArgCount = %d, want 0", cif.ArgCount)
	}
	if cif.ReturnType != nil {
		t.Error("zero CallInterface.ReturnType should be nil")
	}
	if cif.Flags != 0 {
		t.Errorf("zero CallInterface.Flags = %d, want 0", cif.Flags)
	}
	if cif.StackBytes != 0 {
		t.Errorf("zero CallInterface.StackBytes = %d, want 0", cif.StackBytes)
	}
}

func TestCallingConventionConstants(t *testing.T) {
	if UnixCallingConvention != 1 {
		t.Errorf("UnixCallingConvention = %d, want 1", UnixCallingConvention)
	}
	if WindowsCallingConvention != 2 {
		t.Errorf("WindowsCallingConvention = %d, want 2", WindowsCallingConvention)
	}
	if GnuWindowsCallingConvention != 3 {
		t.Errorf("GnuWindowsCallingConvention = %d, want 3", GnuWindowsCallingConvention)
	}
	if DefaultCall != 0 {
		t.Errorf("DefaultCall = %d, want 0", DefaultCall)
	}
}

func TestReturnFlagConstants(t *testing.T) {
	if ReturnVoid != 0 {
		t.Errorf("ReturnVoid = %d, want 0", ReturnVoid)
	}
	if ReturnViaPointer != 1<<10 {
		t.Errorf("ReturnViaPointer = %d, want %d", ReturnViaPointer, 1<<10)
	}
}
