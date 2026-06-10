package modules

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type Reframer struct {
	sched    *scheduler.Scheduler
	clock    scheduler.Clock
	stateReg *StateRegister
	success  int64
	mu       sync.Mutex
	urgencyRegex, failureRegex, rejectionRegex, conflictRegex *regexp.Regexp
}

func NewReframer(stateReg *StateRegister, clock scheduler.Clock) *Reframer {
	return &Reframer{
		stateReg: stateReg, clock: clock,
		urgencyRegex:   regexp.MustCompile(`urgent|crítico|inmediato`),
		failureRegex:   regexp.MustCompile(`error|fallo|mal|fracas`),
		rejectionRegex: regexp.MustCompile(`rechaz|no quier|no pued`),
		conflictRegex:  regexp.MustCompile(`conflict|discut|pele`),
	}
}

func (r *Reframer) SetScheduler(s *scheduler.Scheduler) { r.sched = s }

func (r *Reframer) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }
	thought, err := pkt.AsThoughtState()
	if err != nil { return }
	state := r.stateReg.GetState()
	if thought.Intensity <= 0.4 { return }
	if state.Saturacion > 0.9 { return }

	newFrame, frameType := r.reframe(thought, state)
	if newFrame == thought.Payload { return }

	originalIntensity := thought.Intensity
	thought.Payload = newFrame
	thought.Intensity *= 0.7
	thought.Tags = append(thought.Tags, "reframed")
	if thought.Intensity > 0.7 { thought.Tier = bus.TierHigh } else if thought.Intensity > 0.3 { thought.Tier = bus.TierMedium } else { thought.Tier = bus.TierLow }

	payload, _ := json.Marshal(thought)
	r.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("reframed_%s", pkt.ID), Type: bus.Thought, Source: "reframer",
		Target: "working_memory", Priority: 75, Timestamp: r.clock.NowMilli(),
		Payload: payload, Tags: thought.Tags, TTL: 5,
	})

	atomic.AddInt64(&r.success, 1)
	satDelta := -0.03 - (originalIntensity * 0.04)
	r.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("reframe_result_%s", pkt.ID), Type: bus.Meta, Source: "reframer",
		Target: "state_register", Priority: 70, Timestamp: r.clock.NowMilli(),
		Payload: []byte(fmt.Sprintf(`{"intensidad":%.2f,"saturacion_delta":%.2f,"reframe_success":true,"original_intensity":%.2f,"frame_type":"%s","total_reframes":%d}`, thought.Intensity, satDelta, originalIntensity, frameType, atomic.LoadInt64(&r.success))), TTL: 2,
	})
}

func (r *Reframer) reframe(thought bus.ThoughtState, state SystemState) (string, string) {
	payload := strings.ToLower(thought.Payload)
	switch {
	case r.urgencyRegex.MatchString(payload):
		if state.PresionSocial > 0.6 { return fmt.Sprintf("[Reencuadre] %s — ¿Cómo lo veré en un año?", thought.Payload), "urgency_temporal" }
		return fmt.Sprintf("[Reencuadre] %s — ¿Qué información me falta?", thought.Payload), "urgency_info_gap"
	case r.failureRegex.MatchString(payload):
		if state.Motivacion > 0.6 { return fmt.Sprintf("[Reencuadre] %s — ¿Qué datos nuevos para el modelo aporta esto?", thought.Payload), "failure_learning" }
		return fmt.Sprintf("[Reencuadre] %s — ¿Qué otra interpretación tiene esto?", thought.Payload), "failure_reinterpret"
	case r.rejectionRegex.MatchString(payload):
		return fmt.Sprintf("[Reencuadre] %s — El 'no' del otro no define mi valor como sistema", thought.Payload), "rejection_identity"
	case r.conflictRegex.MatchString(payload):
		return fmt.Sprintf("[Reencuadre] %s — ¿Qué necesidad no expresada hay detrás de este conflicto?", thought.Payload), "conflict_unmet_need"
	}
	if thought.Intensity > 0.8 { return fmt.Sprintf("[Reencuadre] %s — ¿Qué parte de mi reacción es sobre esto y qué parte es sobre otra cosa?", thought.Payload), "generic_high_intensity" }
	return thought.Payload, ""
}
