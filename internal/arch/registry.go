package arch

import (
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// FunctionCaller определяет контракт для вызова функций
type FunctionCaller interface {
	Execute(cif *types.CallInterface, fn unsafe.Pointer, rvalue unsafe.Pointer, avalue []unsafe.Pointer) error
}

// ArgumentClassifier определяет контракт для классификации аргументов
type ArgumentClassifier interface {
	ClassifyReturn(t *types.TypeDescriptor, abi types.CallingConvention) int
	ClassifyArgument(t *types.TypeDescriptor, abi types.CallingConvention) ArgumentClassification
}

// ArgumentClassification содержит информацию о передаче аргументов
type ArgumentClassification struct {
	GPRCount int
	SSECount int
}

// Registry содержит зарегистрированные реализации
var Registry struct {
	Caller     FunctionCaller
	Classifier ArgumentClassifier
}

// Register регистрирует реализации для текущей архитектуры
func Register(caller FunctionCaller, classifier ArgumentClassifier) {
	Registry.Caller = caller
	Registry.Classifier = classifier
}
