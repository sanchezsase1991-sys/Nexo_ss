package modules

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

// PerceptionGate — SECCIÓN 3: Módulo de Producción del Habla (Área de Broca)
// Responsable de:
// 1. Elaboración del plan articulatorio (secuencia de movimientos de salida)
// 2. Codificación de estructura gramatical antes de ejecución
// 3. Monitoreo de salida verbal contra intención comunicativa
// 4. Calibración social del output antes de entrega

type PerceptionGate struct {
	sched       *scheduler.Scheduler
	clock       scheduler.Clock
	stateReg    *StateRegister
	socialCtx   *SocialContextAnalyzer
	wm          *WorkingMemoryManager
	mu          sync.Mutex
	outputQueue []CalibratedOutput
	maxQueue    int
	stats       PerceptionStats
}

type CalibratedOutput struct {
	ID            string
	RawMessage    string
	CalibratedMsg string
	Tone          string
	Intention     string
	Context       string
	Score         float64
	Timestamp     int64
}

type PerceptionStats struct {
	TotalInputs    int
	Accepted       int
	Rejected       int
	Calibrated     int
	SocialFiltered int
	AvgLatency     float64
}

func NewPerceptionGate(stateReg *StateRegister, clock scheduler.Clock, socialCtx *SocialContextAnalyzer, wm *WorkingMemoryManager) *PerceptionGate {
	return &PerceptionGate{
		stateReg:  stateReg,
		clock:     clock,
		socialCtx: socialCtx,
		wm:        wm,
		outputQueue: make([]CalibratedOutput, 0, 64),
		maxQueue:    64,
	}
}

func (pg *PerceptionGate) SetScheduler(s *scheduler.Scheduler) { pg.sched = s }

func (pg *PerceptionGate) Handle(pkt bus.CognitivePacket) {
	switch pkt.Type {
	case bus.Thought:
		pg.processThought(pkt)
	case bus.Action:
		pg.processAction(pkt)
	case bus.Meta:
		pg.processMeta(pkt)
	}
}

func (pg *PerceptionGate) processThought(pkt bus.CognitivePacket) {
	thought, err := pkt.AsThoughtState()
	if err != nil {
		return
	}

	pg.mu.Lock()
	pg.stats.TotalInputs++
	pg.mu.Unlock()

	// FASE 1: Evaluación de señal de entrada
	signalQuality := pg.evaluateSignalQuality(thought)

	// FASE 2: Umbral de percepción
	threshold := pg.getPerceptionThreshold()
	if signalQuality < threshold {
		pg.mu.Lock()
		pg.stats.Rejected++
		pg.mu.Unlock()
		return
	}

	// FASE 3: Formación de percepto
	percept := pg.formPercept(thought, signalQuality)

	// FASE 4: Decisión de acción
	action := pg.decideAction(percept)

	// FASE 5: Emisión de resultado
	pg.emitPerceptionResult(percept, action, pkt)
}

func (pg *PerceptionGate) processAction(pkt bus.CognitivePacket) {
	var actionMsg map[string]any
	json.Unmarshal(pkt.Payload, &actionMsg)

	// Evaluar si el output necesita calibración social
	tone, _ := actionMsg["tone"].(string)
	message, _ := actionMsg["message"].(string)

	if needsCalibration := pg.needsSocialCalibration(tone, message); needsCalibration {
		calibrated := pg.calibrateOutput(message, tone)
		actionMsg["message"] = calibrated.CalibratedMsg
		actionMsg["tone"] = calibrated.Tone
		actionMsg["calibrated"] = true

		pg.mu.Lock()
		pg.stats.Calibrated++
		pg.mu.Unlock()

		payload, _ := json.Marshal(actionMsg)
		pg.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("cal_%s", pkt.ID), Type: bus.Action,
			Source: "perception_gate", Target: "output_formatter",
			Priority: 90, Timestamp: pg.clock.NowMilli(),
			Payload: payload, TTL: 1,
		})
	}
}

func (pg *PerceptionGate) processMeta(pkt bus.CognitivePacket) {
	var meta map[string]any
	json.Unmarshal(pkt.Payload, &meta)
	if statsReq, ok := meta["request_stats"].(bool); ok && statsReq {
		stats := pg.GetStats()
		payload, _ := json.Marshal(stats)
		pg.sched.Emit(bus.CognitivePacket{
			ID: fmt.Sprintf("pg_stats_%d", pg.clock.NowMilli()), Type: bus.Meta,
			Source: "perception_gate", Target: "meta_cognition",
			Priority: 40, Timestamp: pg.clock.NowMilli(),
			Payload: payload, TTL: 3,
		})
	}
}

// evaluateSignalQuality evalúa la calidad de la señal de entrada
func (pg *PerceptionGate) evaluateSignalQuality(thought bus.ThoughtState) float64 {
	score := thought.Score * 0.4
	score += thought.Intensity * 0.3

	// Bonus por relevancia social
	if thought.IsSocial() {
		score += 0.15
	}

	// Bonus por urgencia
	if thought.IsUrgent() {
		score += 0.1
	}

	// Bonus por riqueza de asociaciones
	if thought.HasRichAssociations() {
		score += 0.05
	}

	// Penalización por saturación del sistema
	state := pg.stateReg.GetState()
	score *= (1.0 - state.Saturacion*0.3)

	return clamp(score, 0, 1)
}

// getPerceptionThreshold retorna el umbral de percepción según el estado
func (pg *PerceptionGate) getPerceptionThreshold() float64 {
	state := pg.stateReg.GetState()

	// En estado de flujo, el umbral es más bajo (más receptivo)
	if state.AttentionMode() == AttentionFlow {
		return 0.2
	}

	// En estado degradado, el umbral sube (menos receptivo)
	if state.Saturacion > 0.7 {
		return 0.6
	}

	// Umbral base
	return 0.35
}

// formPercept crea un percepto estructurado a partir del pensamiento
func (pg *PerceptionGate) formPercept(thought bus.ThoughtState, quality float64) Percept {
	return Percept{
		ID:          fmt.Sprintf("percept_%s", thought.OriginalID),
		Source:      thought.Source,
		Content:     thought.Payload,
		Quality:     quality,
		Tags:        thought.Tags,
		IsSocial:    thought.IsSocial(),
		IsUrgent:    thought.IsUrgent(),
		IsQuestion:  thought.IsQuestion(),
		Intensity:   thought.Intensity,
		Timestamp:   pg.clock.NowMilli(),
	}
}

// decideAction determina qué hacer con el percepto
func (pg *PerceptionGate) decideAction(percept Percept) PerceptAction {
	state := pg.stateReg.GetState()

	// Alta prioridad: preguntas urgentes
	if percept.IsQuestion && percept.Intensity > 0.7 {
		return PerceptActionProcess
	}

	// Alta prioridad: contenido social con presión
	if percept.IsSocial && state.PresionSocial > 0.5 {
		return PerceptActionProcess
	}

	// Si el sistema está saturado, solo procesar alta calidad
	if state.Saturacion > 0.7 && percept.Quality < 0.6 {
		return PerceptActionDefer
	}

	// Si hay capacidad, procesar todo con calidad suficiente
	if state.CognitiveCapacity() > 0.4 && percept.Quality > 0.3 {
		return PerceptActionProcess
	}

	return PerceptActionQueue
}

// emitPerceptionResult emite el resultado de la percepción al pipeline
func (pg *PerceptionGate) emitPerceptionResult(percept Percept, action PerceptAction, pkt bus.CognitivePacket) {
	if action == PerceptActionReject {
		return
	}

	result := map[string]any{
		"percept_id": percept.ID,
		"quality":    percept.Quality,
		"action":     action.String(),
		"content":    percept.Content,
		"is_social":  percept.IsSocial,
		"is_urgent":  percept.IsUrgent,
		"intensity":  percept.Intensity,
	}

	payload, _ := json.Marshal(result)

	target := "control_planner"
	priority := 70

	if action == PerceptActionDefer {
		target = "working_memory"
		priority = 40
	}

	pg.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("percept_%s", pkt.ID), Type: bus.Thought,
		Source: "perception_gate", Target: target,
		Priority: priority, Timestamp: pg.clock.NowMilli(),
		Payload: payload, Tags: percept.Tags, TTL: 3,
	})

	pg.mu.Lock()
	pg.stats.Accepted++
	pg.mu.Unlock()
}

// needsSocialCalibration determina si un output necesita calibración social
func (pg *PerceptionGate) needsSocialCalibration(tone, message string) bool {
	// Si el tono es muy intenso, necesita calibración
	if tone == "intense" || tone == "raw" {
		return true
	}

	// Si el mensaje contiene señales de alta carga emocional
	lower := strings.ToLower(message)
	highLoadMarkers := []string{"urgente", "critico", "necesito", "ayuda", "por favor"}
	for _, marker := range highLoadMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}

	// Si el sistema está en estado de alta intensidad
	state := pg.stateReg.GetState()
	if state.Intensidad > 0.7 {
		return true
	}

	return false
}

// calibrateOutput ajusta el output según el contexto social
func (pg *PerceptionGate) calibrateOutput(message, tone string) CalibratedOutput {
	state := pg.stateReg.GetState()

	calibratedTone := tone
	calibratedMsg := message

	// Modulación por saturación
	if state.Saturacion > 0.7 {
		calibratedMsg = truncateStr(message, 100)
		calibratedTone = "minimal"
	}

	// Modulación por intensidad alta
	if state.Intensidad > 0.8 {
		calibratedMsg = "[CALIBRADO] " + calibratedMsg
	}

	// Modulación por presión social alta
	if state.PresionSocial > 0.7 {
		calibratedTone = "social_adapted"
	}

	return CalibratedOutput{
		RawMessage:    message,
		CalibratedMsg: calibratedMsg,
		Tone:          calibratedTone,
		Score:         state.CognitiveCapacity(),
		Timestamp:     pg.clock.NowMilli(),
	}
}

func (pg *PerceptionGate) GetStats() PerceptionStats {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	return pg.stats
}

// Tipos de acción perceptual
type PerceptAction int

const (
	PerceptActionProcess PerceptAction = iota
	PerceptActionDefer
	PerceptActionQueue
	PerceptActionReject
)

func (pa PerceptAction) String() string {
	switch pa {
	case PerceptActionProcess:
		return "PROCESS"
	case PerceptActionDefer:
		return "DEFER"
	case PerceptActionQueue:
		return "QUEUE"
	case PerceptActionReject:
		return "REJECT"
	default:
		return "UNKNOWN"
	}
}

// Percept representa un percepto formado a partir de una señal
type Percept struct {
	ID        string
	Source    string
	Content   string
	Quality   float64
	Tags      []string
	IsSocial  bool
	IsUrgent  bool
	IsQuestion bool
	Intensity float64
	Timestamp int64
}

// PerceptionState representa el estado de la percepción
type PerceptionState int

const (
	PerceptionIdle PerceptionState = iota
	PerceptionReceiving
	PerceptionProcessing
	PerceptionCalibrating
)

// LatencyEstimate estima la latencia del procesamiento perceptual
func (pg *PerceptionGate) LatencyEstimate(state SystemState) float64 {
	base := 10.0 // ms base
	if state.Saturacion > 0.7 {
		base *= 2.0
	}
	if state.Intensidad > 0.8 {
		base *= 1.5
	}
	return base
}
