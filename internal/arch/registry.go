package arch

import (
	"unsafe"

	"github.com/go-webgpu/goffi/types"
)

// FunctionCaller defines the contract for function execution
type FunctionCaller interface {
	Execute(cif *types.CallInterface, fn unsafe.Pointer, rvalue unsafe.Pointer, avalue []unsafe.Pointer) error
}

// ArgumentClassifier defines the contract for argument classification
type ArgumentClassifier interface {
	ClassifyReturn(t *types.TypeDescriptor, abi types.CallingConvention) int
	ClassifyArgument(t *types.TypeDescriptor, abi types.CallingConvention) ArgumentClassification
}

// ArgumentClassification contains argument passing information
type ArgumentClassification struct {
	GPRCount int
	SSECount int
}

// Registry contains registered implementations
var Registry struct {
	Caller     FunctionCaller
	Classifier ArgumentClassifier
}

// Register registers implementations for the current architecture
func Register(caller FunctionCaller, classifier ArgumentClassifier) {
	Registry.Caller = caller
	Registry.Classifier = classifier
}
