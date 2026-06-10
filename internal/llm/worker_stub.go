//go:build !cgo

package llm

import "fmt"

type LLMWorker struct {
	id      int
	purpose string
	busy    bool
}

func NewWorker(id int, modelPath string, ctxSize int, threads int, purpose string) (*LLMWorker, error) {
	return nil, fmt.Errorf("llm worker requires cgo (llama.cpp)")
}

func (w *LLMWorker) Generate(prompt string, maxTokens int, temperature float64) (string, error) {
	return "", fmt.Errorf("llm not available without cgo")
}

func (w *LLMWorker) IsBusy() bool    { return w.busy }
func (w *LLMWorker) ID() int         { return w.id }
func (w *LLMWorker) Purpose() string { return w.purpose }
func (w *LLMWorker) Close()          {}
