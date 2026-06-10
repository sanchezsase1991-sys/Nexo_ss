package modules

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type CoherenceResult struct {
	Alignment     ModuleAlignment
	Coherence     float64
	Conflicts     []string
	Rumiacion     bool
	ShouldInhibit bool
	InhibitReason string
}

type ControlPlanner struct {
	sched            *scheduler.Scheduler
	clock            scheduler.Clock
	stateReg         *StateRegister
	wm               *WorkingMemoryManager
	predictive       *PredictiveSimulator
	socialAnalyzer   *SocialContextAnalyzer
	interpreter      Interpreter
	mu               sync.Mutex
	pendingDecisions []PendingDecision
	decisionTraces   []DecisionTrace
	reframeCount     int
	inhibitCount     int
}

func NewControlPlanner(stateReg *StateRegister, clock scheduler.Clock, interpreter Interpreter) *ControlPlanner {
	return &ControlPlanner{
		stateReg: stateReg, clock: clock, interpreter: interpreter,
		pendingDecisions: make([]PendingDecision, 0, 64),
		decisionTraces:   make([]DecisionTrace, 0, 256),
	}
}

func (cp *ControlPlanner) SetScheduler(s *scheduler.Scheduler)            { cp.sched = s }
func (cp *ControlPlanner) SetPredictiveSimulator(ps *PredictiveSimulator) { cp.predictive = ps }
func (cp *ControlPlanner) SetSocialAnalyzer(sa *SocialContextAnalyzer)    { cp.socialAnalyzer = sa }
func (cp *ControlPlanner) SetWorkingMemory(wm *WorkingMemoryManager)      { cp.wm = wm }

func (cp *ControlPlanner) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }
	thought, err := pkt.AsThoughtState()
	if err != nil { return }
	state := cp.stateReg.GetState()
	impact := cp.wm.CheckOverflow()
	alignment, coherenceScore, conflicts := cp.evaluateAlignment(thought, state, impact)
	result := cp.resolveDecision(thought, state, alignment, coherenceScore, conflicts, impact)
	cp.executeOrDefer(pkt, thought, result, state)
}

func (cp *ControlPlanner) evaluateAlignment(thought bus.ThoughtState, state SystemState, impact CapacityImpact) (ModuleAlignment, float64, []string) {
	var alignment ModuleAlignment
	var conflicts []string
	alignment.Heuristica = cp.evaluateHeuristicAlignment(thought, state)
	alignment.Analisis = cp.evaluateAnalyticAlignment(thought, state)
	alignment.Validacion = cp.evaluateValidationAlignment(thought, state)
	alignment.Trigger = cp.evaluateTriggerAlignment(thought, state)
	coherenceScore := cp.calculateCoherenceScore(alignment, state, impact)
	if alignment.Heuristica != alignment.Analisis { conflicts = append(conflicts, "heuristic_analytic_divergence") }
	if alignment.Validacion && !alignment.Analisis { conflicts = append(conflicts, "values_logic_conflict") }
	if alignment.Trigger && !alignment.Validacion { conflicts = append(conflicts, "urgency_without_purpose") }
	return alignment, coherenceScore, conflicts
}

func (cp *ControlPlanner) evaluateHeuristicAlignment(thought bus.ThoughtState, state SystemState) bool {
	if thought.Intensity > 0.6 && thought.HasRichAssociations() { return true }
	if thought.IsSocial() && state.PresionSocial > 0.4 { return true }
	if thought.IsTemporalDeadline() { return true }
	return thought.Score > 0.5
}

func (cp *ControlPlanner) evaluateAnalyticAlignment(thought bus.ThoughtState, state SystemState) bool {
	if thought.Score > 0.6 && state.CognitiveCapacity() > 0.4 { return true }
	if thought.IsQuestion() && state.CognitiveCapacity() > 0.5 { return true }
	if thought.IsToolRequest() && state.Saturacion < 0.7 { return true }
	return false
}

func (cp *ControlPlanner) evaluateValidationAlignment(thought bus.ThoughtState, state SystemState) bool {
	values := float64(DefaultWeights.RelevanciaValores) / 100.0 * thought.Intensity
	social := float64(DefaultWeights.ImpactoSocial) / 100.0
	justice := float64(DefaultWeights.Justicia) / 100.0
	alignmentScore := 0.0
	if thought.IsSocial() { alignmentScore += social * thought.Intensity }
	if thought.IsFromKnownAgent() { alignmentScore += 0.2 }
	if thought.HasImplicitContent() { alignmentScore += 0.15 * values }
	if thought.IsQuestion() { alignmentScore += justice * 0.3 }
	if thought.IsUrgent() && state.PresionSocial > 0.6 { alignmentScore += 0.2 * values }
	return alignmentScore > 0.3
}

func (cp *ControlPlanner) evaluateTriggerAlignment(thought bus.ThoughtState, state SystemState) bool {
	if thought.IsUrgent() { return true }
	if thought.IsTemporalDeadline() { return true }
	if thought.Intensity > 0.7 && state.Motivacion > 0.6 { return true }
	return state.PresionSocial > 0.7 && thought.IsFromKnownAgent()
}

func (cp *ControlPlanner) calculateCoherenceScore(alignment ModuleAlignment, state SystemState, impact CapacityImpact) float64 {
	convergent := countTrue(alignment.Heuristica, alignment.Analisis, alignment.Validacion, alignment.Trigger)
	var coherence float64
	switch convergent {
	case 4: coherence = 0.95
	case 3: coherence = 0.75
	case 2: coherence = 0.45
	case 1: coherence = 0.20
	default: coherence = 0.05
	}
	coherence *= (1.0 - impact.ExplicitRecallPenalty*0.3)
	coherence *= (0.5 + 0.5*state.Clarity())
	if state.Motivacion > 0.6 && convergent >= 2 { coherence = math.Min(coherence+0.1, 1.0) }
	return clamp(coherence, 0, 1)
}

func (cp *ControlPlanner) resolveDecision(thought bus.ThoughtState, state SystemState, alignment ModuleAlignment, coherenceScore float64, conflicts []string, impact CapacityImpact) DecisionResult {
	input := cp.buildDecisionInput(thought, state)
	score := cp.decisionScore(input)
	phase := cp.getDegradationPhase(state, impact)
	switch {
	case coherenceScore >= 0.8 && phase == PhaseProductiva:
		return DecisionResult{State: DecisionAligned, Score: score, ShouldExecute: true}
	case coherenceScore >= 0.6 && phase != PhaseModoSeguro:
		return DecisionResult{State: DecisionPartialMatch, Score: score, ShouldExecute: true, Rumiacion: true}
	case coherenceScore >= 0.4 && phase == PhaseProductiva:
		return DecisionResult{State: DecisionPendingReassessment, Score: score, Reenqueue: true}
	case coherenceScore >= 0.3 && phase == PhaseDegradacion:
		return DecisionResult{State: DecisionPendingReassessment, Score: score, Rumiacion: true, Reenqueue: true}
	case coherenceScore < 0.3 || phase >= PhaseAgotamiento:
		return DecisionResult{State: DecisionBlocked, Score: score, Deadlock: true}
	default:
		return DecisionResult{State: DecisionPendingReassessment, Score: score, Reenqueue: true}
	}
}

func (cp *ControlPlanner) buildDecisionInput(thought bus.ThoughtState, state SystemState) DecisionInput {
	return DecisionInput{
		UrgenciaExterna: thought.Intensity, ImportanciaObjetiva: thought.Score,
		RelevanciaValores: float64(DefaultWeights.RelevanciaValores) / 100.0,
		ImpactoSocial: float64(DefaultWeights.ImpactoSocial) / 100.0,
		Justicia: float64(DefaultWeights.Justicia) / 100.0,
		Eficiencia: float64(DefaultWeights.Eficiencia) / 100.0,
		Carga: state.Load(),
	}
}

func (cp *ControlPlanner) decisionScore(in DecisionInput) float64 {
	motivacional := (in.RelevanciaValores*0.35 + in.Justicia*0.35) * in.ImpactoSocial
	pragmatico := in.UrgenciaExterna*0.6 + in.ImportanciaObjetiva*0.4
	emergencia := 0.0
	if in.UrgenciaExterna > 0.85 { emergencia = 0.5 }
	raw := motivacional*0.7 + pragmatico*0.2 + emergencia*0.1
	if in.RelevanciaValores < 0.2 && in.UrgenciaExterna > 0.5 { raw = math.Max(raw, 0.3) }
	return clamp(raw-in.Carga*0.15, 0, 1)
}

func (cp *ControlPlanner) getDegradationPhase(state SystemState, impact CapacityImpact) DegradationPhase {
	ccv := cp.calculateCCV(state, impact)
	switch { case ccv > 70: return PhaseProductiva; case ccv > 40: return PhaseDegradacion; case ccv > 20: return PhaseAgotamiento; default: return PhaseModoSeguro }
}

func (cp *ControlPlanner) calculateCCV(state SystemState, impact CapacityImpact) float64 {
	return clamp(CalculateCCV(70.0, state.Motivacion*10.0, state.Stress()*20.0, impact.VelocityMod*5.0, (1.0-state.Saturacion)*10.0), 0, 100)
}

func (cp *ControlPlanner) executeOrDefer(pkt bus.CognitivePacket, thought bus.ThoughtState, result DecisionResult, state SystemState) {
	trace := DecisionTrace{ThoughtID: thought.OriginalID, Urgencia: thought.Intensity, Carga: state.Saturacion, Riesgo: result.Score, Valores: float64(DefaultWeights.RelevanciaValores) / 100.0, Score: result.Score, Decision: result.State.String()}
	cp.mu.Lock()
	cp.decisionTraces = append(cp.decisionTraces, trace)
	if len(cp.decisionTraces) > 256 { cp.decisionTraces = cp.decisionTraces[1:] }
	cp.mu.Unlock()
	if result.ShouldExecute {
		impact := cp.wm.CheckOverflow()
		degraded := impact.SaturationLevel == SaturationHigh || impact.SaturationLevel == SaturationModerate
		collapsed := impact.SaturationLevel == SaturationCritical
		message := cp.interpreter.GenerateMessage(thought, result.State.String(), result.Score)
		tone := cp.interpreter.SelectTone(state, thought)
		actionPayload, _ := json.Marshal(map[string]any{"message": message, "tone": tone, "decision": result.State.String(), "score": result.Score, "degraded_mode": degraded, "collapsed": collapsed, "rumination": result.Rumiacion, "thought_id": thought.OriginalID})
		cp.sched.Emit(bus.CognitivePacket{ID: fmt.Sprintf("dec_%s", pkt.ID), Type: bus.Action, Source: "control_planner", Target: "output_formatter", Priority: 85, Timestamp: cp.clock.NowMilli(), Payload: actionPayload, Tags: append(thought.Tags, result.State.String()), TTL: 5})
	} else if result.Reenqueue {
		cp.mu.Lock()
		cp.pendingDecisions = append(cp.pendingDecisions, PendingDecision{Packet: pkt, Thought: thought, State: state, Timestamp: cp.clock.NowMilli()})
		cp.mu.Unlock()
	} else {
		cp.sched.Emit(bus.CognitivePacket{ID: fmt.Sprintf("blocked_%s", pkt.ID), Type: bus.Meta, Source: "control_planner", Target: "state_register", Priority: 30, Timestamp: cp.clock.NowMilli(), Payload: []byte(`{"saturacion_delta":0.02}`), TTL: 2})
	}
}
