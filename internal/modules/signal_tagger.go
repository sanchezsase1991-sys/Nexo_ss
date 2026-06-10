package modules

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type taggedSignal struct {
	pkt       bus.CognitivePacket
	relevance float64
	tier      string
	tags      []string
	intensity float64
}

type SignalTagger struct {
	sched         *scheduler.Scheduler
	clock         scheduler.Clock
	PerfilWeights prioridadRelevancia
	stateReg      *StateRegister
	highQueue     []taggedSignal
	mediumQueue   []taggedSignal
	lowQueue      []taggedSignal
	mu            sync.Mutex
}

func NewSignalTagger(w prioridadRelevancia, clock scheduler.Clock) *SignalTagger {
	return &SignalTagger{PerfilWeights: w, clock: clock}
}

func (st *SignalTagger) SetScheduler(s *scheduler.Scheduler) { st.sched = s }
func (st *SignalTagger) SetStateRegister(sr *StateRegister)  { st.stateReg = sr }

func (st *SignalTagger) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Perception { return }
	if st.stateReg == nil { return }
	payload := string(pkt.Payload)
	intensity := 0.5
	for _, t := range pkt.Tags { if strings.HasPrefix(t, "intensity:") { fmt.Sscanf(t, "intensity:%f", &intensity) } }
	tags := st.extractTags(payload)
	// Preservar etiquetas originales del paquete (excepto intensity)
	for _, t := range pkt.Tags {
		if !strings.HasPrefix(t, "intensity:") {
			tags = append(tags, t)
		}
	}
	state := st.stateReg.GetState()
	relevance := st.computeRelevance(intensity, state)
	var tier string
	switch {
	case relevance > 0.7: tier = "ALTA"
	case relevance > 0.3: tier = "MEDIA"
	default: tier = "BAJA"
	}
	signal := taggedSignal{pkt: pkt, relevance: relevance, tier: tier, tags: tags, intensity: intensity}
	st.mu.Lock()
	switch tier {
	case "ALTA":
		st.highQueue = append(st.highQueue, signal)
		if len(st.highQueue) > 100 { st.highQueue = st.highQueue[len(st.highQueue)-100:] }
		st.mu.Unlock()
		st.emitSignal(signal)
	case "MEDIA":
		st.mediumQueue = append(st.mediumQueue, signal)
		if len(st.mediumQueue) > 200 { st.mediumQueue = st.mediumQueue[len(st.mediumQueue)-200:] }
		st.mu.Unlock()
		if state.CognitiveCapacity() > 0.5 { st.emitSignal(signal) }
	case "BAJA":
		st.lowQueue = append(st.lowQueue, signal)
		if len(st.lowQueue) > 500 { st.lowQueue = st.lowQueue[len(st.lowQueue)-500:] }
		st.mu.Unlock()
		if state.CognitiveCapacity() > 0.6 { st.emitSignal(signal) }
	}
	// NOTA: El flush automático en recepción causa ciclos de realimentación.
	// Las colas se vacían en segundo plano con temporizadores.
	// if state.CognitiveCapacity() > 0.7 && len(st.lowQueue) > 0 { st.FlushLowQueue() }
	// if state.CognitiveCapacity() > 0.5 && len(st.mediumQueue) > 0 { st.FlushMediumQueue() }
}

func (st *SignalTagger) computeRelevance(intensity float64, state SystemState) float64 {
	wV := float64(st.PerfilWeights.RelevanciaValores) / 100.0
	wS := float64(st.PerfilWeights.ImpactoSocial) / 100.0
	return clamp((0.6*wV+0.4*wS)*intensity*(0.4+0.6*state.CognitiveCapacity()), 0, 1)
}

func (st *SignalTagger) FlushMediumQueue() {
	st.mu.Lock()
	s := append([]taggedSignal(nil), st.mediumQueue...)
	st.mediumQueue = nil
	st.mu.Unlock()
	state := st.stateReg.GetState()
	for _, sig := range s { if state.CognitiveCapacity() > 0.4 { st.emitSignal(sig) } }
}

func (st *SignalTagger) FlushLowQueue() {
	st.mu.Lock()
	s := append([]taggedSignal(nil), st.lowQueue...)
	st.lowQueue = nil
	st.mu.Unlock()
	state := st.stateReg.GetState()
	for _, sig := range s { if state.CognitiveCapacity() > 0.7 { st.emitSignal(sig) } }
}

func (st *SignalTagger) PromoteFromLow(state SystemState) {
	st.mu.Lock()
	promoted := []taggedSignal{}
	remaining := []taggedSignal{}
	for _, sig := range st.lowQueue {
		if sig.intensity > 0.6 || (state.Intensidad > 0.5 && sig.relevance > 0.2) {
			sig.relevance = clamp(sig.relevance*1.5, 0, 1)
			promoted = append(promoted, sig)
		} else {
			remaining = append(remaining, sig)
		}
	}
	st.lowQueue = remaining
	st.mu.Unlock()
	for _, sig := range promoted { st.emitSignal(sig) }
}

func (st *SignalTagger) emitSignal(signal taggedSignal) {
	out, err := json.Marshal(map[string]any{
		"id": signal.pkt.ID, "payload": string(signal.pkt.Payload),
		"relevance_score": signal.relevance, "tier": signal.tier,
		"tags": signal.tags, "intensity": signal.intensity,
	})
	if err != nil { return }
	priority := 85
	switch signal.tier { case "ALTA": priority = 90; case "MEDIA": priority = 70; case "BAJA": priority = 50 }
	st.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("tag_%s", signal.pkt.ID), Type: bus.Thought, Source: "signal_tagger",
		Target: "working_memory", Priority: priority, Timestamp: st.clock.NowMilli(),
		Payload: out, Tags: signal.tags, TTL: 10,
	})
	st.persistToQueue(signal)
}

func (st *SignalTagger) persistToQueue(signal taggedSignal) {
	tagsJSON, _ := json.Marshal(signal.tags)
	record := map[string]interface{}{
		"signal_id": signal.pkt.ID, "payload": string(signal.pkt.Payload),
		"relevance_score": signal.relevance, "tier": signal.tier,
		"tags": string(tagsJSON), "intensity": signal.intensity,
		"source": signal.pkt.Source, "enqueued_at": st.clock.NowMilli(), "processed": false,
	}
	payload, _ := json.Marshal(record)
	st.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("sq_%s", signal.pkt.ID), Type: bus.Meta, Source: "signal_tagger",
		Target: "long_term_memory", Priority: 25, Timestamp: st.clock.NowMilli(),
		Payload: payload, Tags: []string{"signal_queue", signal.tier}, TTL: 5,
	})
}

func (st *SignalTagger) extractTags(payload string) []string {
	l := strings.ToLower(payload)
	t := []string{"tagged"}
	for _, p := range []string{"hola", "buenas", "hey", "adiós", "gracias", "por favor", "disculpa"} {
		if strings.Contains(l, p) { t = append(t, "social"); break }
	}
	for _, p := range []string{"urgente", "ahora", "ya", "inmediato", "crítico", "emergencia"} {
		if strings.Contains(l, p) { t = append(t, "urgent"); break }
	}
	if strings.Contains(l, "?") || strings.HasPrefix(l, "qué") || strings.HasPrefix(l, "cómo") || strings.HasPrefix(l, "cuándo") || strings.HasPrefix(l, "dónde") || strings.HasPrefix(l, "quién") {
		t = append(t, "question")
	}
	for _, p := range []string{"batería", "battery", "ubicación", "gps", "wifi", "cámara", "foto", "linterna", "notificación", "alarma", "configuración"} {
		if strings.Contains(l, p) { t = append(t, "tool_request"); break }
	}
	return t
}
