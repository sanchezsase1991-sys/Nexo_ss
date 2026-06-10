package modules

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type CircuitState int32

const (
	CircuitClosed   CircuitState = iota
	CircuitHalfOpen
	CircuitOpen
)

type CircuitBreaker struct {
	sched          *scheduler.Scheduler
	clock          scheduler.Clock
	state          atomic.Int32
	failureCount   atomic.Int64
	successCount   atomic.Int64
	maxLoad        int32
	currentLoad    atomic.Int32
	mu             sync.Mutex
	lastFailureTime int64
	cooldownPeriod  time.Duration
}

func NewCircuitBreaker(maxLoad int32, cooldownMs int64) *CircuitBreaker {
	cb := &CircuitBreaker{maxLoad: maxLoad, cooldownPeriod: time.Duration(cooldownMs) * time.Millisecond}
	cb.state.Store(int32(CircuitClosed))
	return cb
}

func (cb *CircuitBreaker) SetScheduler(s *scheduler.Scheduler) { cb.sched = s }

func (cb *CircuitBreaker) Allow() bool {
	currentState := CircuitState(cb.state.Load())
	switch currentState {
	case CircuitClosed:
		if cb.currentLoad.Add(1) > cb.maxLoad { cb.currentLoad.Add(-1); cb.trip(); return false }
		return true
	case CircuitHalfOpen:
		if cb.currentLoad.Add(1) == 1 { return true }
		cb.currentLoad.Add(-1)
		return false
	case CircuitOpen:
		cb.mu.Lock()
		elapsed := time.Since(time.UnixMilli(cb.lastFailureTime))
		cb.mu.Unlock()
		if elapsed >= cb.cooldownPeriod { cb.state.Store(int32(CircuitHalfOpen)); return cb.Allow() }
		return false
	default: return false
	}
}

func (cb *CircuitBreaker) Done(success bool) {
	cb.currentLoad.Add(-1)
	if success { cb.successCount.Add(1); cb.failureCount.Store(0); cb.state.Store(int32(CircuitClosed)) } else { cb.failureCount.Add(1); cb.mu.Lock(); cb.lastFailureTime = time.Now().UnixMilli(); cb.mu.Unlock() }
}

func (cb *CircuitBreaker) trip() {
	cb.state.Store(int32(CircuitOpen))
	cb.mu.Lock()
	cb.lastFailureTime = time.Now().UnixMilli()
	cb.mu.Unlock()
	log.Printf("[CircuitBreaker] TRIPPED")
	cb.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("cb_trip_%d", cb.lastFailureTime), Type: bus.Meta,
		Source: "circuit_breaker", Target: "control_planner", Priority: 95,
		Timestamp: cb.lastFailureTime, Payload: []byte(`{"alert":"circuit_breaker_tripped"}`),
		Tags: []string{"circuit_breaker"}, TTL: 3,
	})
}
