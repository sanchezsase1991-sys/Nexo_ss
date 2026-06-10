package modules

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type SystemState struct {
	Valencia, Intensidad, Saturacion, PresionSocial, PresionTemporal, Motivacion float64
	LastUpdate int64
}

func (s SystemState) Stress() float64 { return s.Saturacion*0.6 + s.PresionSocial*0.4 }
func (s SystemState) Clarity() float64 { return (1-s.Saturacion)*(1-s.Intensidad) }
func (s SystemState) Load() float64 { return s.Saturacion }
func (s SystemState) CognitiveCapacity() float64 {
	ns := clamp(s.Saturacion/7.0, 0, 1)
	return clamp((1.0-ns)*0.5+s.Motivacion*0.3+(1.0-ns)*0.2, 0, 1)
}

type AttentionState int
const (AttentionFlow AttentionState = iota; AttentionSustained; AttentionDegrading; AttentionCollapsed)

func (s SystemState) AttentionMode() AttentionState {
	ps := s.Valencia*0.4 + s.Motivacion*0.3 + s.PresionSocial*0.3
	cap := s.CognitiveCapacity()
	switch {
	case ps > 0.7 && cap > 0.6: return AttentionFlow
	case ps > 0.4 && cap > 0.4: return AttentionSustained
	case cap > 0.2: return AttentionDegrading
	default: return AttentionCollapsed
	}
}

type StateRegister struct {
	state SystemState
	clock scheduler.Clock
	mu    sync.RWMutex
	sched *scheduler.Scheduler
	ctx   context.Context
	cancel context.CancelFunc
}

func NewStateRegister(clock scheduler.Clock) *StateRegister {
	return &StateRegister{clock: clock, state: SystemState{Valencia: 0.5, Intensidad: 0.2}}
}

func (sr *StateRegister) SetScheduler(s *scheduler.Scheduler) { sr.sched = s }

func (sr *StateRegister) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Meta:
		var u map[string]float64
		json.Unmarshal(pkt.Payload, &u)
		sr.mu.Lock()
		if v, ok := u["valencia"]; ok { sr.state.Valencia = clamp(v, -1, 1) }
		if v, ok := u["intensidad"]; ok { sr.state.Intensidad = clamp(v, 0, 1) }
		if v, ok := u["saturacion"]; ok { sr.state.Saturacion = clamp(v, 0, 1) }
		if v, ok := u["valencia_delta"]; ok { sr.state.Valencia = clamp(sr.state.Valencia+v, -1, 1) }
		if v, ok := u["intensidad_delta"]; ok { sr.state.Intensidad = clamp(sr.state.Intensidad+v, 0, 1) }
		if v, ok := u["saturacion_delta"]; ok { sr.state.Saturacion = clamp(sr.state.Saturacion+v, 0, 1) }
		sr.state.LastUpdate = sr.clock.NowMilli()
		sr.mu.Unlock()
	case bus.Thought:
		sr.mu.Lock()
		var p map[string]interface{}
		json.Unmarshal(pkt.Payload, &p)
		r := 0.5
		if v, ok := p["relevance_score"].(float64); ok { r = v }
		sr.state.Intensidad = clamp(sr.state.Intensidad+0.02+r*0.06, 0, 1)
		sr.mu.Unlock()
	}
}

func (sr *StateRegister) DecayTick() {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.state.Intensidad *= 0.95
	sr.state.Saturacion *= 0.90
	sr.state.PresionSocial *= 0.95
	sr.state.PresionTemporal *= 0.97
	sr.state.Motivacion *= 0.99
	t := 0.5
	if sr.state.Saturacion > 0.7 { t = 0.3 } else if sr.state.Intensidad > 0.8 { t = 0.4 }
	sr.state.Valencia += (t - sr.state.Valencia) * 0.01
}

func (sr *StateRegister) StartDecay() {
	sr.ctx, sr.cancel = context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-sr.ctx.Done():
				return
			case <-ticker.C:
				sr.DecayTick()
			}
		}
	}()
}

func (sr *StateRegister) Stop() { if sr.cancel != nil { sr.cancel() } }
func (sr *StateRegister) GetState() SystemState { sr.mu.RLock(); defer sr.mu.RUnlock(); return sr.state }
