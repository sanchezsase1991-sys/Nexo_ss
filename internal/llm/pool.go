package llm

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type LLMPool struct {
	workers map[string][]*LLMWorker
	mu      sync.RWMutex
	metrics *PoolMetrics
}

type PoolMetrics struct {
	TotalRequests   int64
	TotalTokens     int64
	AvgLatency      float64
	Errors          int64
	LastRequestTime int64
	mu              sync.Mutex
}

func NewLLMPool() *LLMPool {
	return &LLMPool{
		workers: make(map[string][]*LLMWorker),
		metrics: &PoolMetrics{},
	}
}

func (p *LLMPool) AddWorker(purpose string, worker *LLMWorker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workers[purpose] = append(p.workers[purpose], worker)
	log.Printf("[LLMPool] Worker %d added to pool '%s'", worker.ID(), purpose)
}

func (p *LLMPool) GetWorker(purpose string) (*LLMWorker, error) {
	p.mu.RLock()
	pool, exists := p.workers[purpose]
	p.mu.RUnlock()

	if !exists {
		p.mu.RLock()
		for _, workers := range p.workers {
			for _, w := range workers {
				if !w.IsBusy() {
					p.mu.RUnlock()
					return w, nil
				}
			}
		}
		p.mu.RUnlock()
		return nil, fmt.Errorf("no available workers for purpose '%s'", purpose)
	}

	for _, w := range pool {
		if !w.IsBusy() {
			return w, nil
		}
	}

	return p.waitForWorker(pool)
}

func (p *LLMPool) waitForWorker(pool []*LLMWorker) (*LLMWorker, error) {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for worker")
		case <-ticker.C:
			for _, w := range pool {
				if !w.IsBusy() {
					return w, nil
				}
			}
		}
	}
}

func (p *LLMPool) Generate(purpose, prompt string, maxTokens int, temperature float64) (string, error) {
	worker, err := p.GetWorker(purpose)
	if err != nil {
		p.metrics.mu.Lock()
		p.metrics.Errors++
		p.metrics.mu.Unlock()
		return "", err
	}

	start := time.Now()
	result, err := worker.Generate(prompt, maxTokens, temperature)
	latency := time.Since(start).Milliseconds()

	p.metrics.mu.Lock()
	p.metrics.TotalRequests++
	p.metrics.TotalTokens += int64(maxTokens)
	if p.metrics.TotalRequests > 0 {
		p.metrics.AvgLatency = (p.metrics.AvgLatency*float64(p.metrics.TotalRequests-1) + float64(latency)) / float64(p.metrics.TotalRequests)
	}
	p.metrics.LastRequestTime = time.Now().UnixMilli()
	if err != nil {
		p.metrics.Errors++
	}
	p.metrics.mu.Unlock()

	return result, err
}

func (p *LLMPool) GetMetrics() PoolMetrics {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	return *p.metrics
}

func (p *LLMPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, pool := range p.workers {
		for _, w := range pool {
			w.Close()
		}
	}
	p.workers = make(map[string][]*LLMWorker)
}
