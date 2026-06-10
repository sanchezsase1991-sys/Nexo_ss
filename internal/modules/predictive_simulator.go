package modules

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/sanchezsase1991-sys/Nexo_ss/internal/bus"
	"github.com/sanchezsase1991-sys/Nexo_ss/internal/scheduler"
)

type ConsequenceBranch struct {
	Description string  `json:"description"`
	Probability float64 `json:"probability"`
	Impact      float64 `json:"impact"`
}

type ConsequenceTree struct {
	Level1, Level2, Level3, Level4 []ConsequenceBranch
	Cost                           float64
	Depth                          int
	TotalRisk                      float64
}

type PredictiveSimulator struct {
	sched         *scheduler.Scheduler
	clock         scheduler.Clock
	stateReg      *StateRegister
	PerfilWeights prioridadRelevancia
	history       []ConsequenceTree
	mu            sync.Mutex
}

func NewPredictiveSimulator(stateReg *StateRegister, weights prioridadRelevancia, clock scheduler.Clock) *PredictiveSimulator {
	return &PredictiveSimulator{stateReg: stateReg, PerfilWeights: weights, clock: clock, history: make([]ConsequenceTree, 0, 100)}
}

func (ps *PredictiveSimulator) SetScheduler(s *scheduler.Scheduler) { ps.sched = s }

func (ps *PredictiveSimulator) Handle(pkt bus.CognitivePacket) {
	if pkt.Type != bus.Thought { return }
	thought, err := pkt.AsThoughtState()
	if err != nil { return }
	state := ps.stateReg.GetState()
	tree := ps.simulate(thought, state)
	ps.mu.Lock()
	ps.history = append(ps.history, tree)
	if len(ps.history) > 100 { ps.history = ps.history[1:] }
	ps.mu.Unlock()
	rp, _ := json.Marshal(map[string]interface{}{"thought": thought, "tree": tree})
	ps.sched.Emit(bus.CognitivePacket{
		ID: fmt.Sprintf("sim_%s", pkt.ID), Type: bus.Thought, Source: "predictive_simulator",
		Target: "control_planner", Priority: 75, Timestamp: ps.clock.NowMilli(),
		Payload: rp, Tags: []string{"simulation_result"}, TTL: 3,
	})
}

func (ps *PredictiveSimulator) simulate(thought bus.ThoughtState, state SystemState) ConsequenceTree {
	t := ConsequenceTree{Level1: ps.projectLevel1(thought, state), Level2: ps.projectLevel2(thought, state), Level3: ps.projectLevel3(thought, state), Depth: 3}
	c := ps.calculateSimulationCost(t)
	if ps.shouldProjectLevel4(t, state) { t.Level4 = ps.projectLevel4(thought, state, t); t.Depth = 4 }
	t.Cost = c
	t.TotalRisk = ps.calculateTotalRisk(t)
	return t
}

func (ps *PredictiveSimulator) shouldProjectLevel4(t ConsequenceTree, s SystemState) bool {
	c := ps.calculateSimulationCost(t)
	th := 0.5
	if s.PresionSocial > 0.7 { th += 0.2 }
	if s.Motivacion > 0.8 { th += 0.15 }
	if s.Saturacion > 0.6 { th -= 0.2 }
	return c < clamp(th, 0.2, 0.9)
}

func (ps *PredictiveSimulator) calculateSimulationCost(t ConsequenceTree) float64 {
	return clamp(float64(len(t.Level1))*0.05+float64(len(t.Level2))*0.08+float64(len(t.Level3))*0.12+float64(len(t.Level4))*0.15, 0, 1)
}

func (ps *PredictiveSimulator) calculateTotalRisk(t ConsequenceTree) float64 {
	r := 0.0
	var all []ConsequenceBranch
	all = append(all, t.Level1...)
	all = append(all, t.Level2...)
	all = append(all, t.Level3...)
	all = append(all, t.Level4...)
	for _, b := range all { if b.Impact < 0 { r += math.Abs(b.Impact) * b.Probability } }
	return clamp(r, 0, 1)
}

func (ps *PredictiveSimulator) projectLevel1(thought bus.ThoughtState, state SystemState) []ConsequenceBranch {
	branches := make([]ConsequenceBranch, 0, 4)
	agentID := thought.Source
	if agentID != "" && agentID != "unknown" {
		branches = append(branches,
			ConsequenceBranch{Description: fmt.Sprintf("Reacción probable de agente %s: %s", agentID, ps.predictReaction(agentID, thought)), Probability: 0.8, Impact: 0.3},
			ConsequenceBranch{Description: fmt.Sprintf("Estado resultante del agente %s: %s", agentID, ps.predictState(agentID, thought)), Probability: 0.75, Impact: 0.4},
		)
	}
	switch {
	case thought.IsUrgent(): branches = append(branches, ConsequenceBranch{Description: "Respuesta inmediata con prioridad elevada", Probability: 0.95, Impact: 0.7}, ConsequenceBranch{Description: "Posible interrupción de tareas en curso", Probability: 0.6, Impact: -0.4})
	case thought.IsSocial(): branches = append(branches, ConsequenceBranch{Description: "Fortalecimiento del vínculo de comunicación", Probability: 0.7, Impact: 0.6}, ConsequenceBranch{Description: "Apertura de canal de diálogo", Probability: 0.85, Impact: 0.5}, ConsequenceBranch{Description: "Establecimiento de tono conversacional", Probability: 0.9, Impact: 0.3})
	case thought.IsQuestion(): branches = append(branches, ConsequenceBranch{Description: "Entrega de información solicitada", Probability: 0.9, Impact: 0.5}, ConsequenceBranch{Description: "Posible necesidad de aclaraciones adicionales", Probability: 0.4, Impact: -0.2})
	case thought.IsToolRequest(): branches = append(branches, ConsequenceBranch{Description: "Activación de herramienta externa", Probability: 0.85, Impact: 0.4}, ConsequenceBranch{Description: "Espera de resultado asíncrono", Probability: 0.9, Impact: -0.1})
	default: branches = append(branches, ConsequenceBranch{Description: "Procesamiento estándar de la señal", Probability: 0.95, Impact: 0.1})
	}
	if state.Saturacion > 0.6 { branches = append(branches, ConsequenceBranch{Description: "Mayor latencia por carga del sistema", Probability: 0.7, Impact: -0.5}) }
	return branches
}

func (ps *PredictiveSimulator) projectLevel2(thought bus.ThoughtState, state SystemState) []ConsequenceBranch {
	branches := make([]ConsequenceBranch, 0, 3)
	if thought.IsSocial() { branches = append(branches, ConsequenceBranch{Description: "Modificación del estado de la relación con el agente", Probability: 0.65, Impact: 0.5}, ConsequenceBranch{Description: "Establecimiento de precedente para futuras interacciones", Probability: 0.55, Impact: 0.4}) }
	if thought.IsUrgent() { branches = append(branches, ConsequenceBranch{Description: "Posible fatiga del sistema por modo de alta prioridad sostenido", Probability: 0.5, Impact: -0.6}, ConsequenceBranch{Description: "Reasignación de recursos de tareas de fondo", Probability: 0.7, Impact: -0.3}) }
	if state.Saturacion > 0.5 { branches = append(branches, ConsequenceBranch{Description: "Riesgo de degradación en la calidad de respuestas posteriores", Probability: 0.55, Impact: -0.5}) }
	branches = append(branches, ConsequenceBranch{Description: "Actualización de la memoria de trabajo con el resultado", Probability: 0.9, Impact: 0.2})
	return branches
}

func (ps *PredictiveSimulator) projectLevel3(thought bus.ThoughtState, state SystemState) []ConsequenceBranch {
	branches := make([]ConsequenceBranch, 0, 4)
	vs := float64(ps.PerfilWeights.RelevanciaValores) / 100.0
	js := float64(ps.PerfilWeights.Justicia) / 100.0
	ss := float64(ps.PerfilWeights.ImpactoSocial) / 100.0
	if vs > 0.8 { branches = append(branches, ConsequenceBranch{Description: "Decisión alineada con valores fundamentales del sistema", Probability: 0.9, Impact: 0.8}) } else if vs < 0.4 { branches = append(branches, ConsequenceBranch{Description: "Posible conflicto con valores del núcleo", Probability: 0.6, Impact: -0.7}) }
	branches = append(branches, ConsequenceBranch{Description: fmt.Sprintf("Auto-evaluación: el sistema se verá como %s tras esta decisión", ps.predictSelfEvaluation(vs, js)), Probability: 0.85, Impact: vs * 0.5})
	pi := ps.calculatePrecedentImpact(thought, ss)
	branches = append(branches, ConsequenceBranch{Description: fmt.Sprintf("Precedente establecido: %s", ps.describePrecedent(thought, pi)), Probability: 0.7, Impact: pi})
	branches = append(branches, ConsequenceBranch{Description: "Registro en la memoria episódica", Probability: 0.95, Impact: 0.2})
	return branches
}

func (ps *PredictiveSimulator) projectLevel4(thought bus.ThoughtState, state SystemState, tree ConsequenceTree) []ConsequenceBranch {
	branches := make([]ConsequenceBranch, 0, 2)
	if tree.Cost < 0.5 { branches = append(branches, ConsequenceBranch{Description: "Impacto sistémico mínimo", Probability: 0.9, Impact: 0.05}); return branches }
	if thought.IsUrgent() { branches = append(branches, ConsequenceBranch{Description: "Posible establecimiento de patrón de urgencia", Probability: 0.4, Impact: -0.3}) }
	if state.Saturacion > 0.7 { branches = append(branches, ConsequenceBranch{Description: "Riesgo de colapso en cadena", Probability: 0.35, Impact: -0.8}, ConsequenceBranch{Description: "Necesidad de activar modo de baja carga", Probability: 0.5, Impact: 0.3}) }
	branches = append(branches, ConsequenceBranch{Description: "Actualización de la red semántica", Probability: 0.6, Impact: 0.15})
	return branches
}

func (ps *PredictiveSimulator) predictReaction(agentID string, thought bus.ThoughtState) string {
	if thought.IsUrgent() { return "atención inmediata y posible estrés" }
	if thought.IsSocial() { return "apertura y reciprocidad comunicativa" }
	return "reacción neutra o de espera"
}

func (ps *PredictiveSimulator) predictState(agentID string, thought bus.ThoughtState) string {
	if thought.IsUrgent() { return "alerta elevada, posible ansiedad" }
	if thought.IsSocial() { return "validación social positiva" }
	return "estado estable sin cambios significativos"
}

func (ps *PredictiveSimulator) predictSelfEvaluation(vs, js float64) string {
	switch avg := (vs + js) / 2; {
	case avg > 0.8: return "consistente, íntegro y alineado"
	case avg > 0.5: return "mayormente coherente con algunas reservas"
	default: return "en conflicto parcial con su identidad"
	}
}

func (ps *PredictiveSimulator) calculatePrecedentImpact(thought bus.ThoughtState, ss float64) float64 {
	vs := float64(ps.PerfilWeights.RelevanciaValores) / 100.0
	impact := (ss*0.4)+(vs*0.3)
	if thought.IsUrgent() { impact += 0.2 }
	if thought.IsSocial() { impact += 0.1 }
	return clamp(impact, -1, 1)
}

func (ps *PredictiveSimulator) describePrecedent(_ bus.ThoughtState, impact float64) string {
	switch {
	case impact > 0.6: return "referencia para situaciones similares"
	case impact > 0.3: return "influirá moderadamente"
	default: return "precedente débil"
	}
}

func (ps *PredictiveSimulator) GetRiskScore(tree ConsequenceTree) float64 { return tree.TotalRisk }
