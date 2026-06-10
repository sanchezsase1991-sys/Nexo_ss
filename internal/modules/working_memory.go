package modules

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/memory"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

const (StandardCapacityMin = 4; StandardCapacityMax = 7; NexoCapacityMin = 9; NexoCapacityMax = 12)

type WorkingMemoryManager struct {
	mem    *memory.MemoryBus
	sched  *scheduler.Scheduler
	clock  scheduler.Clock
	chunks int
	mu     sync.Mutex
}

func NewWorkingMemoryManager(mem *memory.MemoryBus, clock scheduler.Clock) *WorkingMemoryManager {
	return &WorkingMemoryManager{mem: mem, clock: clock}
}

func (wm *WorkingMemoryManager) SetScheduler(s *scheduler.Scheduler) { wm.sched = s }

func (wm *WorkingMemoryManager) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Thought:
		var thought bus.ThoughtState
		json.Unmarshal(pkt.Payload, &thought)
		wm.mu.Lock()
		wm.chunks++
		wm.mu.Unlock()
		key := fmt.Sprintf("wm:%s", thought.OriginalID)
		value, _ := json.Marshal(thought)
		wm.mem.Write(key, value, wm.clock.NowMilli())
		sat := wm.calcSaturacion()
		update, _ := json.Marshal(map[string]float64{"saturacion": sat})
		wm.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("wm_sat_%d", wm.clock.NowMilli()), Type: bus.Meta,
			Source: "working_memory", Target: "state_register",
			Priority: clampInt(50+int(sat*40), 50, 90), Timestamp: wm.clock.NowMilli(),
			Payload: update, TTL: 2,
		})
		wm.checkOverflowAndEvict()
	case bus.Memory:
		key := string(pkt.Payload)
		value, found := wm.mem.Read(key)
		if found {
			wm.sched.Emit(bus.CognitivePacket{
				ID: fmt.Sprintf("wm_resp_%s", pkt.ID), Type: bus.Memory,
				Source: "working_memory", Target: pkt.Source, Priority: 60,
				Timestamp: wm.clock.NowMilli(), Payload: value, TTL: 3,
			})
		}
	}
}

func (wm *WorkingMemoryManager) calcSaturacion() float64 {
	wm.mu.Lock()
	ac := wm.chunks
	wm.mu.Unlock()
	if ac < 1 { return 0 }
	capacity := float64(NexoCapacityMax)
	sat := 1.0 - math.Exp(-float64(ac)/(capacity*0.6))
	return clamp(sat, 0, 1)
}

func (wm *WorkingMemoryManager) DecayChunks(state SystemState) {
	var dr float64
	switch {
	case state.Motivacion > 0.7 && state.Saturacion < 0.3: dr = 0.995
	case state.Saturacion > 0.85: dr = 0.85
	case state.Saturacion > 0.75: dr = 0.90
	default: dr = 0.97
	}
	wm.mu.Lock()
	oC := wm.chunks
	wm.chunks = int(float64(wm.chunks) * dr)
	wm.mu.Unlock()
	if oC > 3 && float64(wm.chunks)/float64(clampInt(oC, 1, oC)) < 0.85 {
		wm.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("wm_decay_%d", wm.clock.NowMilli()), Type: bus.Meta,
			Source: "working_memory", Target: "state_register", Priority: 40,
			Timestamp: wm.clock.NowMilli(), Payload: []byte(`{"saturacion_delta":-0.03}`), TTL: 2,
		})
	}
}

func (wm *WorkingMemoryManager) GetChunks() int { wm.mu.Lock(); defer wm.mu.Unlock(); return wm.chunks }

func (wm *WorkingMemoryManager) CheckOverflow() CapacityImpact {
	wm.mu.Lock()
	ac := wm.chunks
	wm.mu.Unlock()
	switch {
	case ac >= NexoCapacityMax: return CapacityImpact{0.4, 0.5, 0.5, SaturationCritical}
	case ac >= NexoCapacityMin: return CapacityImpact{0.6, 0.7, 0.35, SaturationHigh}
	case ac >= StandardCapacityMax: return CapacityImpact{0.8, 0.85, 0.15, SaturationModerate}
	default: return CapacityImpact{1.0, 1.0, 0.0, SaturationNormal}
	}
}

func (wm *WorkingMemoryManager) checkOverflowAndEvict() {
	wm.mu.Lock()
	chunks := wm.chunks
	wm.mu.Unlock()
	if chunks >= NexoCapacityMax {
		wm.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("wm_evict_%d", wm.clock.NowMilli()), Type: bus.Meta,
			Source: "working_memory", Target: "long_term_memory", Priority: 35,
			Timestamp: wm.clock.NowMilli(),
			Payload: []byte(fmt.Sprintf(`{"event":"eviction","chunks_before":%d,"limit":%d}`, chunks, NexoCapacityMax)),
			Tags: []string{"memory_overflow", "eviction"}, TTL: 3,
		})
	}
}
