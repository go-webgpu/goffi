package ffi

import (
	"github.com/go-webgpu/goffi/internal/arch"
	"github.com/go-webgpu/goffi/types"
)

// prepareCallInterfaceCore реализует основную логику подготовки интерфейса вызова
func prepareCallInterfaceCore(
	cif *types.CallInterface,
	convention types.CallingConvention,
	argCount int,
	returnType *types.TypeDescriptor,
	argTypes []*types.TypeDescriptor,
) error {
	// Проверка входных параметров
	if convention <= types.UnixCallingConvention || convention >= types.GnuWindowsCallingConvention {
		return types.ErrUnsupportedCallingConvention
	}

	cif.Convention = convention
	cif.ArgCount = argCount
	cif.ArgTypes = argTypes
	cif.ReturnType = returnType

	// Инициализация составных типов
	if returnType.Size == 0 && returnType.Kind == types.StructType {
		if err := initializeCompositeType(returnType); err != nil {
			return err
		}
	}

	if !isValidType(returnType) {
		return types.ErrInvalidTypeDefinition
	}

	// Вычисление размера стека
	stackBytes := uintptr(0)
	for _, t := range argTypes {
		if t.Size == 0 && t.Kind == types.StructType {
			if err := initializeCompositeType(t); err != nil {
				return err
			}
		}
		if !isValidType(t) {
			return types.ErrInvalidTypeDefinition
		}
		stackBytes = align(stackBytes, t.Alignment)
		stackBytes += align(t.Size, 8)
	}
	cif.StackBytes = stackBytes

	return preparePlatformSpecific(cif)
}

// preparePlatformSpecific выполняет платформо-специфичную подготовку
func preparePlatformSpecific(cif *types.CallInterface) error {
	if arch.Registry.Classifier == nil {
		return types.ErrUnsupportedArchitecture
	}

	cif.Flags = arch.Registry.Classifier.ClassifyReturn(cif.ReturnType, cif.Convention)

	var gprCount, sseCount int
	maxGPR, maxSSE := maxGPRegisters(cif.Convention), maxSSERegisters(cif.Convention)

	for _, arg := range cif.ArgTypes {
		classification := arch.Registry.Classifier.ClassifyArgument(arg, cif.Convention)
		gprCount += classification.GPRCount
		sseCount += classification.SSECount
		if gprCount > maxGPR || sseCount > maxSSE {
			// Обработка переполнения регистров
		}
	}

	// Особенности Windows: требуется 32 байта shadow space
	if cif.Convention == types.WindowsCallingConvention && cif.StackBytes < 32 {
		cif.StackBytes = 32
	}

	return nil
}

// initializeCompositeType инициализирует составной тип
func initializeCompositeType(t *types.TypeDescriptor) error {
	if t == nil || t.Kind != types.StructType || t.Members == nil {
		return types.ErrInvalidTypeDefinition
	}

	t.Size = 0
	t.Alignment = 0

	for _, member := range t.Members {
		if member.Size == 0 && member.Kind == types.StructType {
			if err := initializeCompositeType(member); err != nil {
				return err
			}
		}
		if !isValidType(member) {
			return types.ErrInvalidTypeDefinition
		}

		t.Size = align(t.Size, member.Alignment)
		t.Size += member.Size

		if member.Alignment > t.Alignment {
			t.Alignment = member.Alignment
		}
	}

	t.Size = align(t.Size, t.Alignment)
	return nil
}

// isValidType проверяет корректность описания типа
func isValidType(t *types.TypeDescriptor) bool {
	switch t.Kind {
	case types.VoidType, types.IntType, types.FloatType, types.DoubleType,
		types.UInt8Type, types.SInt8Type, types.UInt16Type, types.SInt16Type,
		types.UInt32Type, types.SInt32Type, types.UInt64Type, types.SInt64Type,
		types.StructType, types.PointerType:
		return true
	default:
		return false
	}
}

// align выравнивает значение по заданной границе
func align(value, alignment uintptr) uintptr {
	return (value + alignment - 1) &^ (alignment - 1)
}

// maxGPRegisters возвращает максимальное количество регистров общего назначения
func maxGPRegisters(convention types.CallingConvention) int {
	if convention == types.UnixCallingConvention {
		return 6 // RDI, RSI, RDX, RCX, R8, R9
	}
	return 4 // Windows: RCX, RDX, R8, R9
}

// maxSSERegisters возвращает максимальное количество SSE регистров
func maxSSERegisters(convention types.CallingConvention) int {
	if convention == types.UnixCallingConvention {
		return 8 // XMM0-7
	}
	return 4 // Windows: XMM0-3
}
