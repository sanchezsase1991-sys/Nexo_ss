package llm

/*
#cgo LDFLAGS: -L${SRCDIR}/../../llama.cpp/build -lllama -lggml -lstdc++ -lm -lpthread
#include "bridge.h"
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

type LLMWorker struct {
	id      int
	ptr     *C.NexoLLM
	purpose string
	mu      sync.Mutex
	busy    bool
}

func NewWorker(id int, modelPath string, ctxSize int, threads int, purpose string) (*LLMWorker, error) {
	cPath := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cPath))

	ptr := C.nexo_llm_init(cPath, C.int(ctxSize), C.int(threads))
	if ptr == nil {
		return nil, fmt.Errorf("worker %d init failed", id)
	}

	return &LLMWorker{
		id:      id,
		ptr:     ptr,
		purpose: purpose,
	}, nil
}

func (w *LLMWorker) Generate(prompt string, maxTokens int, temperature float64) (string, error) {
	w.mu.Lock()
	w.busy = true
	defer func() { w.busy = false; w.mu.Unlock() }()

	cPrompt := C.CString(prompt)
	defer C.free(unsafe.Pointer(cPrompt))

	res := C.nexo_llm_generate(w.ptr, cPrompt, C.int(maxTokens), C.float(temperature))
	if res == nil {
		return "", fmt.Errorf("worker %d generation failed", w.id)
	}
	defer C.nexo_llm_free_response(res)

	return C.GoString(res.text), nil
}

func (w *LLMWorker) IsBusy() bool    { return w.busy }
func (w *LLMWorker) ID() int         { return w.id }
func (w *LLMWorker) Purpose() string { return w.purpose }
func (w *LLMWorker) Close()          { if w.ptr != nil { C.nexo_llm_free(w.ptr); w.ptr = nil } }
