package modules

import "testing"

func TestDecisionScore_NoRelevanceButUrgent(t *testing.T) {
	cp := &ControlPlanner{}
	in := DecisionInput{RelevanciaValores: 0.1, UrgenciaExterna: 0.9, ImpactoSocial: 0.2, Carga: 0.3}
	score := cp.decisionScore(in)
	if score < 0.2 { t.Errorf("Score too low: %.2f", score) }
}

func TestCheckOverflow_Critical(t *testing.T) {
	wm := &WorkingMemoryManager{chunks: 13}
	impact := wm.CheckOverflow()
	if impact.SaturationLevel != SaturationCritical { t.Errorf("Expected critical, got %s", impact.SaturationLevel) }
}
