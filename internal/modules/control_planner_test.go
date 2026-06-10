package modules

import "testing"

func TestEvaluateCoherence_AllAligned(t *testing.T) {
	cp := &ControlPlanner{}
	in := DecisionInput{UrgenciaExterna: 0.8, ImportanciaObjetiva: 0.7, RelevanciaValores: 0.9, ImpactoSocial: 0.8}
	state := SystemState{PresionSocial: 0.7, Motivacion: 0.8}
	result, score := cp.evaluateCoherence(in, state)
	if result.State != DecisionAligned { t.Errorf("Expected Aligned, got %v", result.State) }
	if score < 0.5 { t.Errorf("Score low: %.2f", score) }
}

func TestDecisionScore_NoRelevanceButUrgent(t *testing.T) {
	cp := &ControlPlanner{}
	in := DecisionInput{RelevanciaValores: 0.1, UrgenciaExterna: 0.9, ImpactoSocial: 0.2, Carga: 0.3}
	score := cp.decisionScore(in)
	if score < 0.3 { t.Errorf("Floor failed: %.2f", score) }
}

func TestCheckOverflow_Critical(t *testing.T) {
	wm := &WorkingMemoryManager{chunks: 13}
	impact := wm.CheckOverflow()
	if impact.SaturationLevel != SaturationCritical { t.Errorf("Expected critical, got %s", impact.SaturationLevel) }
}
