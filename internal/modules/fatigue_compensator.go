package modules

import (
	"fmt"
	"sync"
	"time"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// FatiguePhase representa las fases de fatiga
type FatiguePhase int

const (
	FatigueProductive  FatiguePhase = iota // 2-4h: control voluntario funcional
	FatigueDegradation                     // 4-6h: memoria estrecha, calibración imprecisa
	FatigueExhaustion                      // 6h+: colapso inhibitorio, rumiación
	FatigueRecovery                        // Modo seguro requerido
)

func (fp FatiguePhase) String() string {
	return [...]string{"productive", "degradation", "exhaustion", "recovery"}[fp]
}

// FatigueCompensator implementa la compensación de fatiga
type FatigueCompensator struct {
	sched             *scheduler.Scheduler
	clock             scheduler.Clock
	stateReg          *StateRegister
	wm                *WorkingMemoryManager
	mu                sync.Mutex
	phase             FatiguePhase
	effortMinutes     float64
	maxProductiveMin  float64
	maxDegradationMin float64
}

func NewFatigueCompensator(stateReg *StateRegister, wm *WorkingMemoryManager, clock scheduler.Clock) *FatigueCompensator {
	return &FatigueCompensator{
		stateReg:          stateReg,
		wm:                wm,
		clock:             clock,
		phase:             FatigueProductive,
		maxProductiveMin:  240,  // 4 horas
		maxDegradationMin: 360,  // 6 horas
	}
}

func (fc *FatigueCompensator) SetScheduler(s *scheduler.Scheduler) { fc.sched = s }

// Handle procesa señales para monitoreo de fatiga
func (fc *FatigueCompensator) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Meta { return }

	var payload map[string]interface{}
	json.Unmarshal(pkt.Payload, &payload)

	if event, ok := payload["fatigue_event"].(string); ok {
		switch event {
		case "effort_tick":
			fc.effortMinutes++
			fc.updatePhase()
		case "rest_period":
			fc.effortMinutes *= 0.7 // Recuperación parcial
			fc.updatePhase()
		}
	}
}

func (fc *FatigueCompensator) updatePhase() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	state := fc.stateReg.GetState()

	switch {
	case fc.effortMinutes < fc.maxProductiveMin && state.Saturacion < 0.5:
		fc.phase = FatigueProductive
	case fc.effortMinutes < fc.maxDegradationMin && state.Saturacion < 0.75:
		fc.phase = FatigueDegradation
		fc.emitDegradationWarning()
	case state.Saturacion > 0.85:
		fc.phase = FatigueRecovery
		fc.emitRecoveryRequired()
	default:
		fc.phase = FatigueExhaustion
		fc.emitExhaustionAlert()
	}
}

func (fc *FatigueCompensator) emitDegradationWarning() {
	impact := fc.wm.CheckOverflow()

	fc.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("fatigue_degrad_%d", fc.clock.NowMilli()),
		Type:      bus.Meta,
		Source:    "fatigue_compensator",
		Target:    "control_planner",
		Priority:  70,
		Timestamp: fc.clock.NowMilli(),
		Payload: []byte(fmt.Sprintf(`{
			"phase":"%s","effort_minutes":%.0f,"wm_calibration":%.2f,"recommendation":"reduce_load"
		}`, fc.phase.String(), fc.effortMinutes, impact.CalibrationPrecision)),
		Tags: []string{"fatigue", "degradation"},
		TTL:   3,
	})
}

func (fc *FatigueCompensator) emitExhaustionAlert() {
	fc.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("fatigue_exhaust_%d", fc.clock.NowMilli()),
		Type:      bus.Meta,
		Source:    "fatigue_compensator",
		Target:    "state_register",
		Priority:  85,
		Timestamp: fc.clock.NowMilli(),
		Payload:   []byte(`{"saturacion_delta":0.1,"intensidad_delta":0.15,"alert":"exhaustion_imminent"}`),
		Tags:      []string{"fatigue", "exhaustion"},
		TTL:       3,
	})
}

func (fc *FatigueCompensator) emitRecoveryRequired() {
	fc.sched.Emit(bus.CognitivePacket{
		ID:        fmt.Sprintf("fatigue_recovery_%d", fc.clock.NowMilli()),
		Type:      bus.Meta,
		Source:    "fatigue_compensator",
		Target:    "control_planner",
		Priority:  95,
		Timestamp: fc.clock.NowMilli(),
		Payload:   []byte(`{"alert":"recovery_required","mode":"safe_mode","recommendation":"immediate_rest"}`),
		Tags:      []string{"fatigue", "recovery"},
		TTL:       5,
	})
}

func (fc *FatigueCompensator) GetPhase() FatiguePhase {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.phase
}

func (fc *FatigueCompensator) RecordEffort(minutes float64) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.effortMinutes += minutes
	fc.updatePhase()
}

func (fc *FatigueCompensator) RecordRest(minutes float64) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.effortMinutes = max(0, fc.effortMinutes-minutes*2)
	fc.updatePhase()
}
