package modules

import (
	"fmt"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type FatigueCompensator struct {
	sched    *scheduler.Scheduler
	clock    scheduler.Clock
	stateReg *StateRegister
	wm       *WorkingMemoryManager
}

func NewFatigueCompensator(stateReg *StateRegister, wm *WorkingMemoryManager, clock scheduler.Clock) *FatigueCompensator {
	return &FatigueCompensator{stateReg: stateReg, wm: wm, clock: clock}
}

func (fc *FatigueCompensator) SetScheduler(s *scheduler.Scheduler) { fc.sched = s }

func (fc *FatigueCompensator) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Meta { return }
	state := fc.stateReg.GetState()
	if state.Saturacion > 0.7 {
		fc.emitDegradationMode()
	}
	if state.Saturacion > 0.85 {
		fc.emitSafeMode()
	}
}

func (fc *FatigueCompensator) emitDegradationMode() {
	fc.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("fatigue_degrad_%d", fc.clock.NowMilli()), Type: bus.Meta,
		Source: "fatigue_compensator", Target: "control_planner", Priority: 75,
		Timestamp: fc.clock.NowMilli(),
		Payload:   []byte(`{"alert":"degradacion","recomendacion":"simplificar_tareas"}`),
		Tags:      []string{"fatigue", "degradation"}, TTL: 3,
	})
}

func (fc *FatigueCompensator) emitSafeMode() {
	fc.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("fatigue_safe_%d", fc.clock.NowMilli()), Type: bus.Meta,
		Source: "fatigue_compensator", Target: "control_planner", Priority: 90,
		Timestamp: fc.clock.NowMilli(),
		Payload:   []byte(`{"alert":"agotamiento","recomendacion":"modo_seguro"}`),
		Tags:      []string{"fatigue", "safe_mode"}, TTL: 3,
	})
}

func (fc *FatigueCompensator) GetStats() map[string]int {
	return map[string]int{}
}
